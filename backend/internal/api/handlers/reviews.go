package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/service"
)

// ReviewHandler handles the weekly conscious review of missed habit days.
type ReviewHandler struct {
	cfg       *config.Config
	logger    *zap.Logger
	reviewSvc *service.ReviewService
}

func NewReviewHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *ReviewHandler {
	return &ReviewHandler{
		cfg:       cfg,
		logger:    logger,
		reviewSvc: service.NewReviewService(db, logger),
	}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type reviewDayRequest struct {
	ReviewDate string `json:"review_date"` // "YYYY-MM-DD"
	Status     string `json:"status"`      // "migrated" | "discarded"
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// GetMissedDays returns habits with no check-in for the past 7 days (excluding today).
func (h *ReviewHandler) GetMissedDays(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	missed, err := h.reviewSvc.GetMissedDays(r.Context(), userID)
	if err != nil {
		h.logger.Error("get missed days failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to get missed days")
		return
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

	review, err := h.reviewSvc.ReviewDay(r.Context(), habitID, userID, req.ReviewDate, req.Status)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "habit not found")
			return
		}
		h.logger.Error("review day failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to save review")
		return
	}

	respondJSON(w, http.StatusOK, review)
}
