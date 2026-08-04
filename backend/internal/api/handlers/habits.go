package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/service"
)

// HabitHandler handles all habit-related HTTP endpoints.
type HabitHandler struct {
	cfg      *config.Config
	logger   *zap.Logger
	habitSvc *service.HabitService
}

func NewHabitHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *HabitHandler {
	return &HabitHandler{
		cfg:      cfg,
		logger:   logger,
		habitSvc: service.NewHabitService(db, logger),
	}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type createHabitRequest struct {
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Icon         string                `json:"icon"`
	Color        string                `json:"color"`
	Frequency    string                `json:"frequency"`
	TargetDays   []int32               `json:"target_days"`
	TimeOfDay    string                `json:"time_of_day"`
	// Advanced check type fields
	CheckType    string                `json:"check_type"`
	TimerMinutes *int                  `json:"timer_minutes,omitempty"`
	DeadlineTime *string               `json:"deadline_time,omitempty"`
	MetricConfig []service.MetricField `json:"metric_config,omitempty"`
}

type checkInRequest struct {
	Date         string                 `json:"date"`
	Notes        string                 `json:"notes"`
	TimerSeconds *int                   `json:"timer_seconds,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	IsManual     bool                   `json:"is_manual"`
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// List returns all active habits for the authenticated user.
func (h *HabitHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	habits, err := h.habitSvc.ListByUser(r.Context(), userID)
	if err != nil {
		h.logger.Error("list habits failed", zap.Error(err), zap.String("user_id", userID))
		respondError(w, http.StatusInternalServerError, "failed to list habits")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"habits": habits})
}

// Create creates a new habit.
func (h *HabitHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req createHabitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "habit name is required")
		return
	}
	if req.Icon == "" {
		req.Icon = "⭐"
	}
	if req.Color == "" {
		req.Color = "#0071E3"
	}
	if req.Frequency == "" {
		req.Frequency = "daily"
	}
	if len(req.TargetDays) == 0 {
		req.TargetDays = []int32{1, 2, 3, 4, 5, 6, 7}
	}
	if req.CheckType == "" {
		req.CheckType = "simple"
	}

	// Sanitize check_type dependent fields to avoid SQL syntax errors on empty strings
	if req.CheckType != "timed" || (req.TimerMinutes != nil && *req.TimerMinutes <= 0) {
		req.TimerMinutes = nil
	}
	if req.CheckType != "deadline" || (req.DeadlineTime != nil && strings.TrimSpace(*req.DeadlineTime) == "") {
		req.DeadlineTime = nil
	}
	if req.CheckType != "metric" {
		req.MetricConfig = nil
	}

	habit, err := h.habitSvc.Create(r.Context(), userID, req.Name, req.Description, req.Icon, req.Color,
		req.Frequency, req.TimeOfDay, req.CheckType, req.TargetDays,
		req.TimerMinutes, req.DeadlineTime, req.MetricConfig)
	if err != nil {
		h.logger.Error("create habit failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create habit")
		return
	}

	respondJSON(w, http.StatusCreated, habit)
}

// GetByID returns a single habit.
func (h *HabitHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	habitID := chi.URLParam(r, "habitID")

	habit, err := h.habitSvc.GetByID(r.Context(), habitID, userID)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "habit not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get habit")
		return
	}

	respondJSON(w, http.StatusOK, habit)
}

// Update modifies an existing habit.
func (h *HabitHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	habitID := chi.URLParam(r, "habitID")

	var req createHabitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	habit, err := h.habitSvc.Update(r.Context(), habitID, userID, req.Name, req.Description, req.Icon, req.Color, req.Frequency, req.TargetDays)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "habit not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update habit")
		return
	}

	respondJSON(w, http.StatusOK, habit)
}

// Delete soft-deletes a habit.
func (h *HabitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	habitID := chi.URLParam(r, "habitID")

	if err := h.habitSvc.Delete(r.Context(), habitID, userID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete habit")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CheckIn marks a habit as done for a given date with type-specific validation.
func (h *HabitHandler) CheckIn(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	habitID := chi.URLParam(r, "habitID")

	var req checkInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := service.CheckInInput{
		Date:         req.Date,
		Notes:        req.Notes,
		TimerSeconds: req.TimerSeconds,
		StartedAt:    req.StartedAt,
		CompletedAt:  req.CompletedAt,
		Metrics:      req.Metrics,
		IsManual:     req.IsManual,
	}

	log, err := h.habitSvc.CheckIn(r.Context(), habitID, userID, input)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "habit not found")
			return
		}
		// Validation errors (deadline passed, timer too short, etc.)
		h.logger.Warn("check-in validation failed", zap.Error(err))
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, log)
}

// UndoCheckIn removes a check-in for a given date.
func (h *HabitHandler) UndoCheckIn(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	habitID := chi.URLParam(r, "habitID")
	date := r.URL.Query().Get("date")

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	if err := h.habitSvc.UndoCheckIn(r.Context(), habitID, userID, date); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to undo check-in")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Logs returns check-in history for a habit.
func (h *HabitHandler) Logs(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	habitID := chi.URLParam(r, "habitID")

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}

	logs, err := h.habitSvc.GetLogs(r.Context(), habitID, userID, from, to)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get logs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

// Stats returns aggregate statistics for all user habits.
func (h *HabitHandler) Stats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	stats, err := h.habitSvc.GetStats(r.Context(), userID)
	if err != nil {
		h.logger.Error("get stats failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// Heatmap returns daily completion counts for the past 365 days.
func (h *HabitHandler) Heatmap(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	data, err := h.habitSvc.GetHeatmap(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get heatmap data")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"heatmap": data})
}

// Ensure uuid is used somewhere (suppress import error before sqlc gen)
var _ = uuid.New
