package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/appctx"
)

// ReviewService handles the conscious review of missed habit days (Bullet
// Journal spirit: migrate or discard). Since 2026-10-19 (product decision):
// only the last reviewWindowDays undecided days are shown, and days left
// undecided longer than that are discarded by the app, marked auto, never
// overwriting a decision the person made.
type ReviewService struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewReviewService(db *pgxpool.Pool, logger *zap.Logger) *ReviewService {
	return &ReviewService{db: db, logger: logger}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type DayReviewDTO struct {
	ID         string    `json:"id"`
	HabitID    string    `json:"habit_id"`
	ReviewDate string    `json:"review_date"`
	Status     string    `json:"status"`
	ReviewedAt time.Time `json:"reviewed_at"`
	// Auto: discarded by the app after the review window, not by the person.
	Auto bool `json:"auto"`
}

const (
	// reviewWindowDays: how many past days (yesterday backwards) stay open
	// for review.
	reviewWindowDays = 2
	// autoDiscardLookbackDays bounds how far back undecided days are
	// discarded, so the first run on an old account stays small.
	autoDiscardLookbackDays = 30
)

// MissedDayDTO represents a habit+date combination with no log and no review yet.
type MissedDayDTO struct {
	HabitID    string        `json:"habit_id"`
	HabitName  string        `json:"habit_name"`
	HabitIcon  string        `json:"habit_icon"`
	HabitColor string        `json:"habit_color"`
	Date       string        `json:"date"`
	Review     *DayReviewDTO `json:"review,omitempty"`
}

// ─── Methods ─────────────────────────────────────────────────────────────────

// GetMissedDays returns the scheduled habit days of the last
// reviewWindowDays (today excluded — it is still in progress) that have no
// check-in and no decision yet. Older undecided days are discarded first.
func (s *ReviewService) GetMissedDays(ctx context.Context, userID string) ([]MissedDayDTO, error) {
	return s.getMissedDays(ctx, userID, appctx.Today(ctx))
}

// missedDaysCTE lists (habit, day) pairs that were scheduled, after the
// habit existed (in the user's timezone), with no check-in, between $2 - $3
// days and $2 - $4 days.
const missedDaysCTE = `
	WITH date_series AS (
		SELECT generate_series($2::date - make_interval(days => $3), $2::date - make_interval(days => $4), '1 day'::interval)::date AS day
	),
	active_habits AS (
		SELECT id, name, icon, color, target_days, (created_at AT TIME ZONE $5)::date AS created_on
		FROM habits WHERE user_id = $1 AND is_active = true
	),
	combinations AS (
		SELECT ah.id AS habit_id, ah.name, ah.icon, ah.color, ds.day
		FROM active_habits ah CROSS JOIN date_series ds
		WHERE EXTRACT(ISODOW FROM ds.day) = ANY(ah.target_days) AND ds.day >= ah.created_on
	),
	missed AS (
		SELECT c.* FROM combinations c
		LEFT JOIN habit_logs hl ON hl.habit_id = c.habit_id AND hl.logged_date = c.day
		WHERE hl.id IS NULL
	)`

// getMissedDays anchors the windows on today, the user's local calendar day
// (not the database server's CURRENT_DATE).
func (s *ReviewService) getMissedDays(ctx context.Context, userID string, today time.Time) ([]MissedDayDTO, error) {
	day := today.Format("2006-01-02")
	tz := appctx.UserTimezone(ctx).String()

	// Past the window and still undecided: the app discards it (auto). A
	// decision the person already made is kept (DO NOTHING).
	if _, err := s.db.Exec(ctx, missedDaysCTE+`
		INSERT INTO habit_day_reviews (habit_id, user_id, review_date, status, auto)
		SELECT habit_id, $1, day, 'discarded', true FROM missed
		ON CONFLICT (habit_id, review_date) DO NOTHING`,
		userID, day, autoDiscardLookbackDays, reviewWindowDays+1, tz,
	); err != nil {
		return nil, fmt.Errorf("auto-discarding old days: %w", err)
	}

	rows, err := s.db.Query(ctx, missedDaysCTE+`
		SELECT m.habit_id::text, m.name, m.icon, m.color, m.day::text
		FROM missed m
		LEFT JOIN habit_day_reviews r ON r.habit_id = m.habit_id AND r.review_date = m.day
		WHERE r.id IS NULL
		ORDER BY m.day DESC, m.name ASC`,
		userID, day, reviewWindowDays, 1, tz,
	)
	if err != nil {
		return nil, fmt.Errorf("querying missed days: %w", err)
	}
	defer rows.Close()

	missed := []MissedDayDTO{}
	for rows.Next() {
		var m MissedDayDTO
		if err := rows.Scan(&m.HabitID, &m.HabitName, &m.HabitIcon, &m.HabitColor, &m.Date); err != nil {
			return nil, fmt.Errorf("scanning missed day: %w", err)
		}
		missed = append(missed, m)
	}
	return missed, rows.Err()
}

// ReviewDay creates or updates a conscious review for a missed day.
// Ownership of the habit is checked first — reviews can never point at
// another user's habit.
func (s *ReviewService) ReviewDay(ctx context.Context, habitID, userID, reviewDate, status string) (*DayReviewDTO, error) {
	var exists bool
	if err := s.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM habits WHERE id = $1 AND user_id = $2 AND is_active = true)",
		habitID, userID,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("checking habit ownership: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	var review DayReviewDTO
	err := s.db.QueryRow(ctx, `
		INSERT INTO habit_day_reviews (habit_id, user_id, review_date, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (habit_id, review_date)
		DO UPDATE SET status = EXCLUDED.status, reviewed_at = NOW(), auto = false
		RETURNING id::text, habit_id::text, review_date::text, status, reviewed_at, auto`,
		habitID, userID, reviewDate, status,
	).Scan(&review.ID, &review.HabitID, &review.ReviewDate, &review.Status, &review.ReviewedAt, &review.Auto)
	if err != nil {
		return nil, fmt.Errorf("saving review: %w", err)
	}
	return &review, nil
}
