package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/appctx"
)

// ReviewService handles the weekly conscious review of missed habit days
// (Bullet Journal spirit: a day without a log is neutral until the user
// decides to migrate or discard it).
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
}

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

// GetMissedDays returns scheduled habit days with no check-in over the past
// 7 days (excluding today — today is still in progress, not "missed").
func (s *ReviewService) GetMissedDays(ctx context.Context, userID string) ([]MissedDayDTO, error) {
	return s.getMissedDays(ctx, userID, appctx.Today(ctx))
}

// getMissedDays anchors the 7-day window on today, the user's local calendar
// day (not the database server's CURRENT_DATE).
func (s *ReviewService) getMissedDays(ctx context.Context, userID string, today time.Time) ([]MissedDayDTO, error) {
	rows, err := s.db.Query(ctx, `
		WITH date_series AS (
			SELECT generate_series(
				$2::date - INTERVAL '7 days',
				$2::date - INTERVAL '1 day',
				'1 day'::interval
			)::date AS day
		),
		active_habits AS (
			SELECT id, name, icon, color, target_days
			FROM habits
			WHERE user_id = $1 AND is_active = true
		),
		combinations AS (
			SELECT ah.id AS habit_id, ah.name, ah.icon, ah.color, ds.day
			FROM active_habits ah
			CROSS JOIN date_series ds
			WHERE EXTRACT(ISODOW FROM ds.day) = ANY(ah.target_days)
		)
		SELECT
			c.habit_id::text, c.name, c.icon, c.color, c.day::text,
			r.id::text, r.status, r.reviewed_at
		FROM combinations c
		LEFT JOIN habit_logs hl ON hl.habit_id = c.habit_id AND hl.logged_date = c.day
		LEFT JOIN habit_day_reviews r ON r.habit_id = c.habit_id AND r.review_date = c.day
		WHERE hl.id IS NULL
		ORDER BY c.day DESC, c.name ASC
	`, userID, today.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("querying missed days: %w", err)
	}
	defer rows.Close()

	var missed []MissedDayDTO
	for rows.Next() {
		var m MissedDayDTO
		var reviewID, reviewStatus *string
		var reviewedAt *time.Time

		if err := rows.Scan(
			&m.HabitID, &m.HabitName, &m.HabitIcon, &m.HabitColor, &m.Date,
			&reviewID, &reviewStatus, &reviewedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning missed day: %w", err)
		}

		if reviewID != nil {
			m.Review = &DayReviewDTO{
				ID:         *reviewID,
				HabitID:    m.HabitID,
				ReviewDate: m.Date,
				Status:     *reviewStatus,
				ReviewedAt: *reviewedAt,
			}
		}

		missed = append(missed, m)
	}
	if missed == nil {
		missed = []MissedDayDTO{}
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
		DO UPDATE SET status = EXCLUDED.status, reviewed_at = NOW()
		RETURNING id::text, habit_id::text, review_date::text, status, reviewed_at`,
		habitID, userID, reviewDate, status,
	).Scan(&review.ID, &review.HabitID, &review.ReviewDate, &review.Status, &review.ReviewedAt)
	if err != nil {
		return nil, fmt.Errorf("saving review: %w", err)
	}
	return &review, nil
}
