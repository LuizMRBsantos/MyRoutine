package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/appctx"
	"github.com/myroutine/backend/internal/service"
)

// WeeklyGoalHandler serves /weekly-goals: simple goals for one week.
type WeeklyGoalHandler struct {
	logger *zap.Logger
	svc    *service.WeeklyGoalService
}

func NewWeeklyGoalHandler(db *pgxpool.Pool, logger *zap.Logger) *WeeklyGoalHandler {
	return &WeeklyGoalHandler{logger: logger, svc: service.NewWeeklyGoalService(db)}
}

// dayOrToday: ?date=YYYY-MM-DD (any day of the week), default today in the
// user's timezone.
func dayOrToday(r *http.Request, given string) string {
	if given != "" {
		return given
	}
	return appctx.Today(r.Context()).Format("2006-01-02")
}

// GET /weekly-goals?date=YYYY-MM-DD
func (h *WeeklyGoalHandler) List(w http.ResponseWriter, r *http.Request) {
	goals, err := h.svc.List(r.Context(), middleware.GetUserID(r.Context()), dayOrToday(r, r.URL.Query().Get("date")))
	if err != nil {
		if errors.Is(err, service.ErrInvalidWeeklyGoal) {
			respondError(w, http.StatusBadRequest, "date must be YYYY-MM-DD")
			return
		}
		h.logger.Error("list weekly goals", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to load weekly goals")
		return
	}
	respondJSON(w, http.StatusOK, goals)
}

// POST /weekly-goals {"title": "...", "date": "YYYY-MM-DD" (optional)}
func (h *WeeklyGoalHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
		Date  string `json:"date"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	goal, err := h.svc.Create(r.Context(), middleware.GetUserID(r.Context()), body.Title, dayOrToday(r, body.Date))
	if err != nil {
		if errors.Is(err, service.ErrInvalidWeeklyGoal) {
			respondError(w, http.StatusBadRequest, "invalid weekly goal")
			return
		}
		h.logger.Error("create weekly goal", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create weekly goal")
		return
	}
	respondJSON(w, http.StatusCreated, goal)
}

// PATCH /weekly-goals/{id} {"done": true|false}
func (h *WeeklyGoalHandler) SetDone(w http.ResponseWriter, r *http.Request) {
	id, ok := goalIDParam(w, r)
	if !ok {
		return
	}
	var body struct {
		Done *bool `json:"done"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil || body.Done == nil {
		respondError(w, http.StatusBadRequest, "done is required")
		return
	}
	goal, err := h.svc.SetDone(r.Context(), id, middleware.GetUserID(r.Context()), *body.Done)
	if err != nil {
		h.notFoundOr500(w, err, "update weekly goal")
		return
	}
	respondJSON(w, http.StatusOK, goal)
}

// DELETE /weekly-goals/{id}
func (h *WeeklyGoalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := goalIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id, middleware.GetUserID(r.Context())); err != nil {
		h.notFoundOr500(w, err, "delete weekly goal")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// goalIDParam: a malformed id is simply "not found", never a database error.
func goalIDParam(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		respondError(w, http.StatusNotFound, "weekly goal not found")
		return "", false
	}
	return id, true
}

func (h *WeeklyGoalHandler) notFoundOr500(w http.ResponseWriter, err error, action string) {
	if errors.Is(err, service.ErrNotFound) {
		respondError(w, http.StatusNotFound, "weekly goal not found")
		return
	}
	h.logger.Error(action, zap.Error(err))
	respondError(w, http.StatusInternalServerError, "failed to save weekly goal")
}
