package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Web Push: which devices get notifications (push_subscriptions) and what
// each person wants to receive (notification_settings). Sending lives in
// notifications_dispatch.go.

var (
	// ErrInvalidSubscription: not a browser PushSubscription from a known
	// push service.
	ErrInvalidSubscription = errors.New("invalid push subscription")
	// ErrInvalidSettings: a time or lead outside the allowed values.
	ErrInvalidSettings = errors.New("invalid notification settings")
)

// pushServiceHosts are the official push services. The server POSTs to the
// subscription endpoint when sending, so accepting any URL would let anyone
// make MyRoutine call arbitrary addresses (SSRF).
var pushServiceHosts = []string{
	"web.push.apple.com",                // Safari (iPhone, iPad, Mac)
	"fcm.googleapis.com",                // Chrome, Edge, Android
	"updates.push.services.mozilla.com", // Firefox
	".notify.windows.com",               // Edge on Windows (wns2-*.notify.windows.com)
}

var base64URL = regexp.MustCompile(`^[A-Za-z0-9_-]+={0,2}$`)

type NotificationService struct {
	db *pgxpool.Pool
}

func NewNotificationService(db *pgxpool.Pool) *NotificationService {
	return &NotificationService{db: db}
}

// PushSubscriptionInput is the browser's PushSubscription.toJSON() shape.
type PushSubscriptionInput struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// NotificationSettings: what to send, in the person's local time.
type NotificationSettings struct {
	TaskReminders   bool   `json:"task_reminders"`
	TaskLeadMinutes int    `json:"task_lead_minutes"`
	MorningDigest   bool   `json:"morning_digest"`
	MorningTime     string `json:"morning_time"` // "HH:MM"
	EveningDigest   bool   `json:"evening_digest"`
	EveningTime     string `json:"evening_time"` // "HH:MM"
}

// DefaultNotificationSettings are the product defaults (decided 2026-10-08):
// appointment reminders 15 min before and the 07:00 digest on; 21:00 off.
var DefaultNotificationSettings = NotificationSettings{
	TaskReminders:   true,
	TaskLeadMinutes: 15,
	MorningDigest:   true,
	MorningTime:     "07:00",
	EveningDigest:   false,
	EveningTime:     "21:00",
}

// NotificationView is the settings plus how many devices are subscribed.
type NotificationView struct {
	Settings NotificationSettings `json:"settings"`
	Devices  int                  `json:"devices"`
}

func validSubscription(in PushSubscriptionInput) bool {
	if len(in.Endpoint) > 1024 || len(in.Keys.P256dh) > 256 || len(in.Keys.Auth) > 128 {
		return false
	}
	if !base64URL.MatchString(in.Keys.P256dh) || !base64URL.MatchString(in.Keys.Auth) {
		return false
	}
	u, err := url.Parse(in.Endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, allowed := range pushServiceHosts {
		if host == allowed || (strings.HasPrefix(allowed, ".") && strings.HasSuffix(host, allowed)) {
			return true
		}
	}
	return false
}

// Subscribe registers this device for the user. The same device logging in
// as someone else moves to them (the endpoint is unique per device).
func (s *NotificationService) Subscribe(ctx context.Context, userID string, in PushSubscriptionInput, userAgent string) error {
	if !validSubscription(in) {
		return ErrInvalidSubscription
	}
	if len(userAgent) > 512 {
		userAgent = userAgent[:512]
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning subscribe: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth, user_agent)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		 ON CONFLICT (endpoint) DO UPDATE SET
		   user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh,
		   auth = EXCLUDED.auth, user_agent = EXCLUDED.user_agent`,
		userID, in.Endpoint, in.Keys.P256dh, in.Keys.Auth, userAgent,
	); err != nil {
		return fmt.Errorf("saving subscription: %w", err)
	}
	// First device: the person starts with the product defaults.
	if _, err := tx.Exec(ctx,
		`INSERT INTO notification_settings (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		userID,
	); err != nil {
		return fmt.Errorf("creating default settings: %w", err)
	}
	return tx.Commit(ctx)
}

// Unsubscribe removes one of the user's devices. Unknown endpoints are a
// no-op (the device may already be gone).
func (s *NotificationService) Unsubscribe(ctx context.Context, userID, endpoint string) error {
	_, err := s.db.Exec(ctx,
		"DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2", userID, endpoint)
	if err != nil {
		return fmt.Errorf("removing subscription: %w", err)
	}
	return nil
}

// Get returns the user's settings (defaults if never saved) and device count.
func (s *NotificationService) Get(ctx context.Context, userID string) (*NotificationView, error) {
	view := &NotificationView{Settings: DefaultNotificationSettings}

	var got NotificationSettings
	err := s.db.QueryRow(ctx,
		`SELECT task_reminders, task_lead_minutes,
		        morning_digest, to_char(morning_time, 'HH24:MI'),
		        evening_digest, to_char(evening_time, 'HH24:MI')
		 FROM notification_settings WHERE user_id = $1`,
		userID,
	).Scan(&got.TaskReminders, &got.TaskLeadMinutes,
		&got.MorningDigest, &got.MorningTime, &got.EveningDigest, &got.EveningTime)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("loading notification settings: %w", err)
	default:
		view.Settings = got
	}

	if err := s.db.QueryRow(ctx,
		"SELECT count(*) FROM push_subscriptions WHERE user_id = $1", userID,
	).Scan(&view.Devices); err != nil {
		return nil, fmt.Errorf("counting devices: %w", err)
	}
	return view, nil
}

// Update saves the user's settings after validating times and lead.
func (s *NotificationService) Update(ctx context.Context, userID string, in NotificationSettings) (*NotificationView, error) {
	morning, errM := time.Parse("15:04", in.MorningTime)
	evening, errE := time.Parse("15:04", in.EveningTime)
	if errM != nil || errE != nil || in.TaskLeadMinutes < 5 || in.TaskLeadMinutes > 240 {
		return nil, ErrInvalidSettings
	}

	if _, err := s.db.Exec(ctx,
		`INSERT INTO notification_settings
		   (user_id, task_reminders, task_lead_minutes, morning_digest, morning_time, evening_digest, evening_time, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET
		   task_reminders = EXCLUDED.task_reminders, task_lead_minutes = EXCLUDED.task_lead_minutes,
		   morning_digest = EXCLUDED.morning_digest, morning_time = EXCLUDED.morning_time,
		   evening_digest = EXCLUDED.evening_digest, evening_time = EXCLUDED.evening_time,
		   updated_at = NOW()`,
		userID, in.TaskReminders, in.TaskLeadMinutes, in.MorningDigest,
		morning.Format("15:04"), in.EveningDigest, evening.Format("15:04"),
	); err != nil {
		return nil, fmt.Errorf("saving notification settings: %w", err)
	}
	return s.Get(ctx, userID)
}
