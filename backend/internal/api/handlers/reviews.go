package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/config"
)

// ReviewHandler handles the weekly conscious review of missed habit days.
type ReviewHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	db     *pgxpool.Pool
}

func NewReviewHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *ReviewHandler {
	return &ReviewHandler{cfg: cfg, logger: logger, db: db}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type reviewDayRequest struct {
	HabitID    string `json:"habit_id"`
	ReviewDate string `json:"review_date"` // "YYYY-MM-DD"
	Status     string `json:"status"`      // "migrated" | "discarded"
}

type DayReviewDTO struct {
	ID         string    `json:"id"`
	HabitID    string    `json:"habit_id"`
	ReviewDate string    `json:"review_date"`
	Status     string    `json:"status"`
	ReviewedAt time.Time `json:"reviewed_at"`
}

// MissedDayDTO represents a habit+date combination with no log and no review yet.
type MissedDayDTO struct {
	HabitID   string `json:"habit_id"`
	HabitName string `json:"habit_name"`
	HabitIcon string `json:"habit_icon"`
	HabitColor string `json:"habit_color"`
	Date      string `json:"date"`
	Review    *DayReviewDTO `json:"review,omitempty"`
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// GetMissedDays returns habits with no check-in for the past 7 days (excluding today).
func (h *ReviewHandler) GetMissedDays(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	rows, err := h.db.Query(r.Context(), `
		WITH date_series AS (
			SELECT generate_series(
				CURRENT_DATE - INTERVAL '7 days',
				CURRENT_DATE - INTERVAL '1 day',
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
			WHERE EXTRACT(DOW FROM ds.day) = ANY(
				SELECT unnest(ah.target_days) % 7
			)
		)
		SELECT
			c.habit_id::text, c.name, c.icon, c.color, c.day::text,
			r.id::text, r.status, r.reviewed_at
		FROM combinations c
		LEFT JOIN habit_logs hl ON hl.habit_id = c.habit_id AND hl.logged_date = c.day
		LEFT JOIN habit_day_reviews r ON r.habit_id = c.habit_id AND r.review_date = c.day
		WHERE hl.id IS NULL
		ORDER BY c.day DESC, c.name ASC
	`, userID)
	if err != nil {
		h.logger.Error("get missed days failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to get missed days")
		return
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
			h.logger.Error("scan missed day", zap.Error(err))
			continue
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
	respondJSON(w, http.StatusOK, map[string]any{"missed_days": missed})
}

// ReviewDay creates or updates a conscious review for a missed day.
func (h *ReviewHandler) ReviewDay(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	habitID := chi.URLParam(r, "habitID")

	var req reviewDayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Status != "migrated" && req.Status != "discarded" {
		respondError(w, http.StatusBadRequest, "status must be 'migrated' or 'discarded'")
		return
	}

	if req.ReviewDate == "" {
		req.ReviewDate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	}

	var review DayReviewDTO
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO habit_day_reviews (id, habit_id, user_id, review_date, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (habit_id, review_date)
		DO UPDATE SET status = EXCLUDED.status, reviewed_at = NOW()
		RETURNING id::text, habit_id::text, review_date::text, status, reviewed_at
	`,
		uuid.New().String(), habitID, userID, req.ReviewDate, req.Status,
	).Scan(&review.ID, &review.HabitID, &review.ReviewDate, &review.Status, &review.ReviewedAt)

	if err != nil {
		h.logger.Error("review day failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to save review")
		return
	}

	respondJSON(w, http.StatusOK, review)
}
