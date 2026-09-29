package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Track Day (diário): one free-text entry per user per day. The entry is the
// source of the events it mentions — an expense or a workout recognized in a
// line becomes a transaction / habit_log with source_type='track_day' and a
// source_id derived from (entry id, line). The event lives in its own module;
// the journal only points to it (cross-module-data-flow).

var (
	// ErrInvalidDate: the date is not YYYY-MM-DD.
	ErrInvalidDate = errors.New("invalid date")
	// ErrJournalTooLong: content above the 20k-character limit.
	ErrJournalTooLong = errors.New("journal entry too long")
	// ErrAlreadyLogged: the habit already has a check-in that day from
	// somewhere else; the journal never overwrites it.
	ErrAlreadyLogged = errors.New("habit already checked in that day")
	// ErrCheckInRejected wraps a habit rule refusing the check-in (e.g. a
	// deadline that already passed); its message is safe to show.
	ErrCheckInRejected = errors.New("check-in rejected")
)

const (
	journalSource      = "track_day"
	journalMaxLength   = 20000
	JournalKindExpense = "transaction"
	JournalKindWorkout = "workout"
)

type JournalService struct {
	db      *pgxpool.Pool
	finance *FinanceService
	habits  *HabitService
}

func NewJournalService(db *pgxpool.Pool, logger *zap.Logger) *JournalService {
	return &JournalService{
		db:      db,
		finance: NewFinanceService(db, logger),
		habits:  NewHabitService(db, logger),
	}
}

// JournalItem is an event registered from the journal on that day.
type JournalItem struct {
	SourceID    string         `json:"source_id"`
	Kind        string         `json:"kind"` // transaction | workout
	Label       string         `json:"label"`
	AmountCents *int64         `json:"amount_cents,omitempty"`
	HabitID     *string        `json:"habit_id,omitempty"`
	Metrics     map[string]any `json:"metrics,omitempty"`
}

type JournalDTO struct {
	Date      string     `json:"date"`
	Content   string     `json:"content"`
	UpdatedAt *time.Time `json:"updated_at"`
	// LineIDs maps each (normalized) non-empty line of the content to the
	// source_id an event registered from it carries.
	LineIDs map[string]string `json:"line_ids"`
	Items   []JournalItem     `json:"items"`
}

// JournalLineKey normalizes a line the same way the web app does (trim and
// collapse whitespace), so editing spacing does not orphan a registration.
func JournalLineKey(line string) string {
	return strings.Join(strings.Fields(line), " ")
}

// journalSourceID is deterministic: the same line of the same entry always
// yields the same id, which makes registering idempotent.
func journalSourceID(entryID uuid.UUID, line string) string {
	return uuid.NewSHA1(entryID, []byte(JournalLineKey(line))).String()
}

func parseJournalDate(date string) (string, error) {
	d, err := time.Parse(dateLayout, date)
	if err != nil {
		return "", ErrInvalidDate
	}
	return d.Format(dateLayout), nil
}

// Get returns the day's entry (empty when none yet) and what was registered.
func (s *JournalService) Get(ctx context.Context, userID, date string) (*JournalDTO, error) {
	date, err := parseJournalDate(date)
	if err != nil {
		return nil, err
	}

	out := &JournalDTO{Date: date, LineIDs: map[string]string{}, Items: []JournalItem{}}
	var entryID uuid.UUID
	var updatedAt time.Time
	err = s.db.QueryRow(ctx,
		`SELECT id, content, updated_at FROM journal_entries WHERE user_id = $1 AND entry_date = $2`,
		userID, date,
	).Scan(&entryID, &out.Content, &updatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// No entry yet: nothing written, nothing derived.
	case err != nil:
		return nil, fmt.Errorf("loading journal: %w", err)
	default:
		out.UpdatedAt = &updatedAt
		for _, line := range strings.Split(out.Content, "\n") {
			if key := JournalLineKey(line); key != "" {
				out.LineIDs[key] = journalSourceID(entryID, key)
			}
		}
	}

	items, err := s.itemsOn(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	out.Items = items
	return out, nil
}

// itemsOn lists the journal-sourced events of the day, from their own tables.
func (s *JournalService) itemsOn(ctx context.Context, userID, date string) ([]JournalItem, error) {
	rows, err := s.db.Query(ctx,
		`SELECT source_id::text, 'transaction', description, amount_cents, NULL::text, NULL::jsonb
		   FROM transactions
		  WHERE user_id = $1 AND source_type = $3 AND occurred_on = $2::date AND source_id IS NOT NULL
		 UNION ALL
		 SELECT hl.source_id::text, 'workout', h.name, NULL::bigint, h.id::text, hl.metrics
		   FROM habit_logs hl JOIN habits h ON h.id = hl.habit_id AND h.user_id = hl.user_id
		  WHERE hl.user_id = $1 AND hl.source_type = $3 AND hl.logged_date = $2::date AND hl.source_id IS NOT NULL`,
		userID, date, journalSource,
	)
	if err != nil {
		return nil, fmt.Errorf("listing journal items: %w", err)
	}
	defer rows.Close()

	items := []JournalItem{}
	for rows.Next() {
		var it JournalItem
		if err := rows.Scan(&it.SourceID, &it.Kind, &it.Label, &it.AmountCents, &it.HabitID, &it.Metrics); err != nil {
			return nil, fmt.Errorf("scanning journal item: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// Save stores the day's text (autosave) and returns the refreshed view.
func (s *JournalService) Save(ctx context.Context, userID, date, content string) (*JournalDTO, error) {
	date, err := parseJournalDate(date)
	if err != nil {
		return nil, err
	}
	if len([]rune(content)) > journalMaxLength {
		return nil, ErrJournalTooLong
	}
	if _, err := s.ensureEntry(ctx, userID, date, &content); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, date)
}

// ensureEntry returns the day's entry id, creating it if needed. With a
// non-nil content it also writes the text.
func (s *JournalService) ensureEntry(ctx context.Context, userID, date string, content *string) (uuid.UUID, error) {
	var id uuid.UUID
	var err error
	if content != nil {
		err = s.db.QueryRow(ctx,
			`INSERT INTO journal_entries (user_id, entry_date, content) VALUES ($1, $2, $3)
			 ON CONFLICT (user_id, entry_date) DO UPDATE SET content = EXCLUDED.content, updated_at = NOW()
			 RETURNING id`,
			userID, date, *content,
		).Scan(&id)
	} else {
		err = s.db.QueryRow(ctx,
			`INSERT INTO journal_entries (user_id, entry_date) VALUES ($1, $2)
			 ON CONFLICT (user_id, entry_date) DO UPDATE SET user_id = EXCLUDED.user_id
			 RETURNING id`,
			userID, date,
		).Scan(&id)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("saving journal entry: %w", err)
	}
	return id, nil
}

// RegisterExpenseInput / RegisterWorkoutInput carry what the web parser read
// from a line. The line itself identifies the item (see journalSourceID).
type RegisterExpenseInput struct {
	AmountCents int64  `json:"amount_cents"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type RegisterWorkoutInput struct {
	HabitID     string         `json:"habit_id"`
	Metrics     map[string]any `json:"metrics"`
	TimeMinutes *int           `json:"time_minutes"`
}

// RegisterExpense turns a journal line into a transaction of that day.
// Registering the same line again updates it instead of duplicating.
func (s *JournalService) RegisterExpense(ctx context.Context, userID, date, line string, in RegisterExpenseInput) (*JournalDTO, error) {
	date, err := parseJournalDate(date)
	if err != nil {
		return nil, err
	}
	entryID, err := s.ensureEntry(ctx, userID, date, nil)
	if err != nil {
		return nil, err
	}
	sourceID := journalSourceID(entryID, line)

	if _, err := s.finance.CreateTransaction(ctx, userID, CreateTransactionInput{
		AmountCents: in.AmountCents,
		Kind:        "expense",
		Category:    in.Category,
		Description: in.Description,
		OccurredOn:  date,
		SourceType:  journalSource,
		SourceID:    &sourceID,
	}); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, date)
}

// RegisterWorkout checks in the chosen habit from a journal line. It never
// overwrites a check-in that came from elsewhere (manual, a task, a study
// session): that one keeps its notes/timer and the journal reports it.
func (s *JournalService) RegisterWorkout(ctx context.Context, userID, date, line string, in RegisterWorkoutInput) (*JournalDTO, error) {
	date, err := parseJournalDate(date)
	if err != nil {
		return nil, err
	}
	if err := requireOwned(ctx, s.db, ownedHabits, in.HabitID, userID); err != nil {
		return nil, err
	}
	entryID, err := s.ensureEntry(ctx, userID, date, nil)
	if err != nil {
		return nil, err
	}
	sourceID := journalSourceID(entryID, line)

	var existingSource string
	var existingID *string
	err = s.db.QueryRow(ctx,
		`SELECT source_type, source_id::text FROM habit_logs
		 WHERE habit_id = $1 AND user_id = $2 AND logged_date = $3::date`,
		in.HabitID, userID, date,
	).Scan(&existingSource, &existingID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("checking existing check-in: %w", err)
	case existingSource != journalSource || existingID == nil || *existingID != sourceID:
		return nil, ErrAlreadyLogged
	}

	// Timed habits need a duration; the journal line's minutes count as a
	// manual entry.
	seconds := 0
	if in.TimeMinutes != nil && *in.TimeMinutes > 0 {
		seconds = *in.TimeMinutes * 60
	}
	if _, err := s.habits.CheckIn(ctx, in.HabitID, userID, CheckInInput{
		Date:         date,
		Metrics:      in.Metrics,
		IsManual:     true,
		TimerSeconds: &seconds,
		SourceType:   journalSource,
		SourceID:     &sourceID,
	}); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", ErrCheckInRejected, err.Error())
	}
	return s.Get(ctx, userID, date)
}

// Unregister undoes an event registered from the journal (by its source id),
// wherever it landed. Only journal-sourced rows of this user are touched.
func (s *JournalService) Unregister(ctx context.Context, userID, date, sourceID string) (*JournalDTO, error) {
	date, err := parseJournalDate(date)
	if err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(sourceID); err != nil {
		return nil, ErrNotFound
	}

	tag, err := s.db.Exec(ctx,
		`DELETE FROM transactions WHERE user_id = $1 AND source_type = $2 AND source_id = $3`,
		userID, journalSource, sourceID,
	)
	if err != nil {
		return nil, fmt.Errorf("undoing expense: %w", err)
	}
	if tag.RowsAffected() == 0 {
		tag, err = s.db.Exec(ctx,
			`DELETE FROM habit_logs WHERE user_id = $1 AND source_type = $2 AND source_id = $3`,
			userID, journalSource, sourceID,
		)
		if err != nil {
			return nil, fmt.Errorf("undoing workout: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return nil, ErrNotFound
		}
	}
	return s.Get(ctx, userID, date)
}
