package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Dispatch is the "carteiro": called every few minutes by the scheduler, it
// decides what each person should receive right now, in their own timezone,
// and sends it. Product rules (product-constitution.md): one consolidated
// notification per moment, never one per habit, and nothing when there is
// nothing useful to say.

// PushTarget is one subscribed device.
type PushTarget struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// PushMessage is the JSON the service worker receives and shows.
type PushMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag,omitempty"`
}

// PushSender delivers an encrypted payload to a device. gone reports that the
// push service no longer knows the subscription (404/410), so it is deleted.
type PushSender interface {
	Send(ctx context.Context, target PushTarget, payload []byte) (gone bool, err error)
}

// DispatchResult counts what one run did.
type DispatchResult struct {
	Sent    int `json:"sent"`
	Removed int `json:"removed"`
	Failed  int `json:"failed"`
}

// digestWindow: a digest still goes out if the scheduler runs a little late
// (or skipped a tick) after the chosen time.
const digestWindow = 30 * time.Minute

// deliveriesKeptFor: past deliveries are only needed to avoid repeats.
// Pruned by the database clock (sent_at is written by it).
const deliveriesKeptFor = 30 * 24 * time.Hour

type notifyRecipient struct {
	userID   string
	loc      *time.Location
	settings NotificationSettings
}

// Dispatch sends whatever is due at now. Each notification is claimed in
// notification_deliveries before sending, so overlapping or repeated runs
// never send it twice (a failed send is not retried: missing one reminder is
// better than spamming).
func (s *NotificationService) Dispatch(ctx context.Context, push PushSender, now time.Time) (DispatchResult, error) {
	var res DispatchResult

	recipients, err := s.recipients(ctx)
	if err != nil {
		return res, err
	}

	var errs []error
	for _, r := range recipients {
		msgs, err := s.dueMessages(ctx, r, now)
		if err != nil {
			errs = append(errs, fmt.Errorf("user %s: %w", r.userID, err))
			continue
		}
		for _, msg := range msgs {
			if err := s.deliver(ctx, push, r.userID, msg, &res); err != nil {
				errs = append(errs, fmt.Errorf("user %s: %w", r.userID, err))
			}
		}
	}

	if _, err := s.db.Exec(ctx,
		"DELETE FROM notification_deliveries WHERE sent_at < NOW() - $1::interval", fmt.Sprintf("%d hours", int(deliveriesKeptFor.Hours())),
	); err != nil {
		errs = append(errs, fmt.Errorf("pruning deliveries: %w", err))
	}
	return res, errors.Join(errs...)
}

// recipients are the active users with at least one subscribed device.
func (s *NotificationService) recipients(ctx context.Context) ([]notifyRecipient, error) {
	rows, err := s.db.Query(ctx,
		`SELECT u.id::text, u.timezone,
		        ns.task_reminders, ns.task_lead_minutes,
		        ns.morning_digest, to_char(ns.morning_time, 'HH24:MI'),
		        ns.evening_digest, to_char(ns.evening_time, 'HH24:MI')
		 FROM notification_settings ns
		 JOIN users u ON u.id = ns.user_id
		 WHERE u.is_active
		   AND EXISTS (SELECT 1 FROM push_subscriptions p WHERE p.user_id = u.id)`)
	if err != nil {
		return nil, fmt.Errorf("loading recipients: %w", err)
	}
	defer rows.Close()

	var out []notifyRecipient
	for rows.Next() {
		var r notifyRecipient
		var tz string
		st := &r.settings
		if err := rows.Scan(&r.userID, &tz, &st.TaskReminders, &st.TaskLeadMinutes,
			&st.MorningDigest, &st.MorningTime, &st.EveningDigest, &st.EveningTime); err != nil {
			return nil, fmt.Errorf("scanning recipient: %w", err)
		}
		r.loc, err = time.LoadLocation(tz)
		if err != nil {
			r.loc = saoPauloLocation
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

var saoPauloLocation = func() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic("notifications: cannot load America/Sao_Paulo: " + err.Error())
	}
	return loc
}()

func (s *NotificationService) dueMessages(ctx context.Context, r notifyRecipient, now time.Time) ([]PushMessage, error) {
	local := now.In(r.loc)
	today := local.Format(dateLayout)
	var msgs []PushMessage

	if r.settings.TaskReminders {
		msg, err := s.taskReminder(ctx, r, now, local)
		if err != nil {
			return nil, err
		}
		if msg != nil {
			msgs = append(msgs, *msg)
		}
	}

	if r.settings.MorningDigest && inDigestWindow(local, r.settings.MorningTime) {
		claimed, err := s.claim(ctx, r.userID, "morning_digest", today)
		if err != nil {
			return nil, err
		}
		if claimed {
			msg, err := s.morningDigest(ctx, r.userID, today)
			if err != nil {
				return nil, err
			}
			if msg != nil {
				msgs = append(msgs, *msg)
			}
		}
	}

	if r.settings.EveningDigest && inDigestWindow(local, r.settings.EveningTime) {
		claimed, err := s.claim(ctx, r.userID, "evening_digest", today)
		if err != nil {
			return nil, err
		}
		if claimed {
			tomorrow := local.AddDate(0, 0, 1).Format(dateLayout)
			msg, err := s.eveningDigest(ctx, r.userID, today, tomorrow)
			if err != nil {
				return nil, err
			}
			if msg != nil {
				msgs = append(msgs, *msg)
			}
		}
	}
	return msgs, nil
}

// inDigestWindow: local is at or up to digestWindow after hhmm that day.
func inDigestWindow(local time.Time, hhmm string) bool {
	clock, err := time.Parse("15:04", hhmm)
	if err != nil {
		return false
	}
	start := time.Date(local.Year(), local.Month(), local.Day(), clock.Hour(), clock.Minute(), 0, 0, local.Location())
	return !local.Before(start) && local.Before(start.Add(digestWindow))
}

// claim records that a notification is being sent; false means it already was.
func (s *NotificationService) claim(ctx context.Context, userID, kind, ref string) (bool, error) {
	tag, err := s.db.Exec(ctx,
		`INSERT INTO notification_deliveries (user_id, kind, ref) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, kind, ref) DO NOTHING`,
		userID, kind, ref)
	if err != nil {
		return false, fmt.Errorf("claiming %s: %w", kind, err)
	}
	return tag.RowsAffected() == 1, nil
}

type upcomingTask struct {
	id, title, date, start string
}

// taskReminder: appointments (tasks with a start time, not finished) that
// start within the lead time, consolidated into one notification.
func (s *NotificationService) taskReminder(ctx context.Context, r notifyRecipient, now, local time.Time) (*PushMessage, error) {
	today := local.Format(dateLayout)
	tomorrow := local.AddDate(0, 0, 1).Format(dateLayout) // a lead can cross midnight
	rows, err := s.db.Query(ctx,
		`SELECT id::text, title, date::text, to_char(start_time, 'HH24:MI')
		 FROM tasks
		 WHERE user_id = $1 AND date IN ($2::date, $3::date)
		   AND start_time IS NOT NULL AND status IN ('planned', 'in_progress')
		 ORDER BY date, start_time, title`,
		r.userID, today, tomorrow)
	if err != nil {
		return nil, fmt.Errorf("loading upcoming tasks: %w", err)
	}
	var candidates []upcomingTask
	for rows.Next() {
		var t upcomingTask
		if err := rows.Scan(&t.id, &t.title, &t.date, &t.start); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		candidates = append(candidates, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	lead := time.Duration(r.settings.TaskLeadMinutes) * time.Minute
	var due []upcomingTask
	for _, t := range candidates {
		startsAt, err := time.ParseInLocation("2006-01-02 15:04", t.date+" "+t.start, r.loc)
		if err != nil || !startsAt.After(now) || startsAt.After(now.Add(lead)) {
			continue
		}
		// The ref includes the time, so moving the appointment re-arms it.
		claimed, err := s.claim(ctx, r.userID, "task_reminder", t.id+"@"+t.date+" "+t.start)
		if err != nil {
			return nil, err
		}
		if claimed {
			due = append(due, t)
		}
	}

	switch len(due) {
	case 0:
		return nil, nil
	case 1:
		return &PushMessage{Title: due[0].title, Body: "Começa às " + due[0].start + ".", URL: "/planner"}, nil
	default:
		parts := make([]string, len(due))
		for i, t := range due {
			parts[i] = t.start + " " + t.title
		}
		return &PushMessage{Title: "Próximos compromissos", Body: strings.Join(parts, " · "), URL: "/planner"}, nil
	}
}

// dayTasks counts the day's unfinished tasks and the earliest start time.
func (s *NotificationService) dayTasks(ctx context.Context, userID, date string) (int, string, error) {
	var n int
	var first *string
	err := s.db.QueryRow(ctx,
		`SELECT count(*), to_char(min(start_time), 'HH24:MI')
		 FROM tasks WHERE user_id = $1 AND date = $2::date AND status IN ('planned', 'in_progress')`,
		userID, date,
	).Scan(&n, &first)
	if err != nil {
		return 0, "", fmt.Errorf("counting tasks: %w", err)
	}
	if first == nil {
		return n, "", nil
	}
	return n, *first, nil
}

func tasksPhrase(n int, first, when string) string {
	phrase := fmt.Sprintf("%d %s %s", n, plural(n, "compromisso", "compromissos"), when)
	if first != "" {
		phrase += ", o primeiro às " + first
	}
	return phrase
}

func (s *NotificationService) morningDigest(ctx context.Context, userID, today string) (*PushMessage, error) {
	tasks, first, err := s.dayTasks(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	var habits int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM habits
		 WHERE user_id = $1 AND is_active
		   AND EXTRACT(ISODOW FROM $2::date)::int = ANY(target_days)`,
		userID, today,
	).Scan(&habits); err != nil {
		return nil, fmt.Errorf("counting habits: %w", err)
	}

	var parts []string
	if tasks > 0 {
		parts = append(parts, tasksPhrase(tasks, first, "hoje"))
	}
	if habits > 0 {
		parts = append(parts, fmt.Sprintf("%d %s para hoje", habits, plural(habits, "hábito", "hábitos")))
	}
	if len(parts) == 0 {
		return nil, nil
	}
	return &PushMessage{Title: "Bom dia", Body: strings.Join(parts, " · ") + ".", URL: "/", Tag: "morning-digest"}, nil
}

func (s *NotificationService) eveningDigest(ctx context.Context, userID, today, tomorrow string) (*PushMessage, error) {
	var open int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM habits h
		 WHERE h.user_id = $1 AND h.is_active
		   AND EXTRACT(ISODOW FROM $2::date)::int = ANY(h.target_days)
		   AND NOT EXISTS (SELECT 1 FROM habit_logs l WHERE l.habit_id = h.id AND l.logged_date = $2::date)`,
		userID, today,
	).Scan(&open); err != nil {
		return nil, fmt.Errorf("counting open habits: %w", err)
	}
	tasks, first, err := s.dayTasks(ctx, userID, tomorrow)
	if err != nil {
		return nil, err
	}

	var parts []string
	if open > 0 {
		parts = append(parts, fmt.Sprintf("%d %s sem registro hoje", open, plural(open, "hábito", "hábitos")))
	}
	if tasks > 0 {
		parts = append(parts, tasksPhrase(tasks, first, "amanhã"))
	}
	if len(parts) == 0 {
		return nil, nil
	}
	return &PushMessage{Title: "Fim do dia", Body: strings.Join(parts, " · ") + ".", URL: "/", Tag: "evening-digest"}, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// deliver sends msg to every device of the user, forgetting devices the push
// service says are gone. A device that fails for another reason is counted
// and skipped; the run goes on.
func (s *NotificationService) deliver(ctx context.Context, push PushSender, userID string, msg PushMessage, res *DispatchResult) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encoding notification: %w", err)
	}

	rows, err := s.db.Query(ctx,
		"SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("loading devices: %w", err)
	}
	var targets []PushTarget
	for rows.Next() {
		var t PushTarget
		if err := rows.Scan(&t.Endpoint, &t.P256dh, &t.Auth); err != nil {
			rows.Close()
			return fmt.Errorf("scanning device: %w", err)
		}
		targets = append(targets, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, t := range targets {
		gone, err := push.Send(ctx, t, payload)
		switch {
		case gone:
			if _, err := s.db.Exec(ctx,
				"DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2", userID, t.Endpoint); err != nil {
				return fmt.Errorf("removing expired device: %w", err)
			}
			res.Removed++
		case err != nil:
			res.Failed++
		default:
			res.Sent++
			if _, err := s.db.Exec(ctx,
				"UPDATE push_subscriptions SET last_used_at = NOW() WHERE user_id = $1 AND endpoint = $2", userID, t.Endpoint); err != nil {
				return fmt.Errorf("marking device used: %w", err)
			}
		}
	}
	return nil
}
