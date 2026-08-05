package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/service"
)

type TaskHandler struct {
	cfg     *config.Config
	logger  *zap.Logger
	taskSvc *service.TaskService
}

func NewTaskHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{
		cfg:     cfg,
		logger:  logger,
		taskSvc: service.NewTaskService(db),
	}
}

// Allowed values for free-text columns (schema documents them only in comments,
// there is no CHECK constraint — the API is the gate).
var (
	validTaskStatuses   = map[string]bool{"planned": true, "in_progress": true, "done": true, "reviewed": true}
	validTaskPriorities = map[string]bool{"high": true, "medium": true, "low": true}
	validGoalStatuses   = map[string]bool{"active": true, "done": true, "abandoned": true}
)

// ─── Tasks ───────────────────────────────────────────────────────────────────

// GET /tasks?date=YYYY-MM-DD
func (h *TaskHandler) ListByDate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	date := r.URL.Query().Get("date")
	if date == "" {
		respondError(w, http.StatusBadRequest, "date query param required")
		return
	}

	tasks, err := h.taskSvc.ListByDate(r.Context(), userID, date)
	if err != nil {
		h.logger.Error("list tasks by date", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

// GET /tasks/week?start=YYYY-MM-DD&end=YYYY-MM-DD
func (h *TaskHandler) ListByWeek(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	if start == "" || end == "" {
		respondError(w, http.StatusBadRequest, "start and end query params required")
		return
	}

	tasks, err := h.taskSvc.ListByWeek(r.Context(), userID, start, end)
	if err != nil {
		h.logger.Error("list tasks by week", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

// POST /tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input service.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Title == "" || input.Date == "" {
		respondError(w, http.StatusBadRequest, "title and date are required")
		return
	}
	if input.Priority != "" && !validTaskPriorities[input.Priority] {
		respondError(w, http.StatusBadRequest, "priority must be high, medium or low")
		return
	}

	task, err := h.taskSvc.Create(r.Context(), userID, input)
	if err != nil {
		h.logger.Error("create task", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	respondJSON(w, http.StatusCreated, task)
}

// PATCH /tasks/{id}
// Body semantics: absent fields are kept, explicit null clears the column.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	var fields map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(fields) == 0 {
		respondError(w, http.StatusBadRequest, "no fields to update")
		return
	}
	if raw, ok := fields["status"]; ok {
		var status string
		if err := json.Unmarshal(raw, &status); err != nil || !validTaskStatuses[status] {
			respondError(w, http.StatusBadRequest, "status must be planned, in_progress, done or reviewed")
			return
		}
	}
	if raw, ok := fields["priority"]; ok {
		var priority string
		if err := json.Unmarshal(raw, &priority); err != nil || !validTaskPriorities[priority] {
			respondError(w, http.StatusBadRequest, "priority must be high, medium or low")
			return
		}
	}

	task, err := h.taskSvc.Update(r.Context(), taskID, userID, fields)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error("update task", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// POST /tasks/{id}/advance
// Avança o status: planned → in_progress → done → reviewed
func (h *TaskHandler) AdvanceStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	task, err := h.taskSvc.AdvanceStatus(r.Context(), taskID, userID)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error("advance task status", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to advance task status")
		return
	}

	respondJSON(w, http.StatusOK, task)
}

// DELETE /tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	if err := h.taskSvc.Delete(r.Context(), taskID, userID); err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error("delete task", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── Monthly Goals ───────────────────────────────────────────────────────────

// GET /goals?month=YYYY-MM-DD
func (h *TaskHandler) ListGoals(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	month := r.URL.Query().Get("month")
	if month == "" {
		respondError(w, http.StatusBadRequest, "month query param required")
		return
	}

	goals, err := h.taskSvc.ListGoalsByMonth(r.Context(), userID, month)
	if err != nil {
		h.logger.Error("list monthly goals", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list goals")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"goals": goals})
}

// POST /goals
func (h *TaskHandler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var body struct {
		Title string  `json:"title"`
		Month string  `json:"month"`
		Notes *string `json:"notes"`
		Color *string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Title == "" || body.Month == "" {
		respondError(w, http.StatusBadRequest, "title and month are required")
		return
	}

	goal, err := h.taskSvc.CreateGoal(r.Context(), userID, body.Title, body.Month, body.Notes, body.Color)
	if err != nil {
		h.logger.Error("create monthly goal", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create goal")
		return
	}

	respondJSON(w, http.StatusCreated, goal)
}

// PATCH /goals/{id}/status
func (h *TaskHandler) UpdateGoalStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	goalID := chi.URLParam(r, "id")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validGoalStatuses[body.Status] {
		respondError(w, http.StatusBadRequest, "status must be active, done or abandoned")
		return
	}

	goal, err := h.taskSvc.UpdateGoalStatus(r.Context(), goalID, userID, body.Status)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "goal not found")
			return
		}
		h.logger.Error("update goal status", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to update goal status")
		return
	}

	respondJSON(w, http.StatusOK, goal)
}

// DELETE /goals/{id}
func (h *TaskHandler) DeleteGoal(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	goalID := chi.URLParam(r, "id")

	if err := h.taskSvc.DeleteGoal(r.Context(), goalID, userID); err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "goal not found")
			return
		}
		h.logger.Error("delete monthly goal", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to delete goal")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
