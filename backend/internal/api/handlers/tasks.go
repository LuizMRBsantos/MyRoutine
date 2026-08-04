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

// ─── Tasks ───────────────────────────────────────────────────────────────────

// GET /tasks?date=YYYY-MM-DD
func (h *TaskHandler) ListByDate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, `{"error":"date query param required"}`, http.StatusBadRequest)
		return
	}

	tasks, err := h.taskSvc.ListByDate(r.Context(), userID, date)
	if err != nil {
		h.logger.Error("list tasks by date", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GET /tasks/week?start=YYYY-MM-DD&end=YYYY-MM-DD
func (h *TaskHandler) ListByWeek(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	if start == "" || end == "" {
		http.Error(w, `{"error":"start and end query params required"}`, http.StatusBadRequest)
		return
	}

	tasks, err := h.taskSvc.ListByWeek(r.Context(), userID, start, end)
	if err != nil {
		h.logger.Error("list tasks by week", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// POST /tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input service.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if input.Title == "" || input.Date == "" {
		http.Error(w, `{"error":"title and date are required"}`, http.StatusBadRequest)
		return
	}

	task, err := h.taskSvc.Create(r.Context(), userID, input)
	if err != nil {
		h.logger.Error("create task", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// PATCH /tasks/{id}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	var input service.UpdateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	task, err := h.taskSvc.Update(r.Context(), taskID, userID, input)
	if err != nil {
		h.logger.Error("update task", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// POST /tasks/{id}/advance
// Avança o status: planned → in_progress → done → reviewed
func (h *TaskHandler) AdvanceStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	task, err := h.taskSvc.AdvanceStatus(r.Context(), taskID, userID)
	if err != nil {
		h.logger.Error("advance task status", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// DELETE /tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	if err := h.taskSvc.Delete(r.Context(), taskID, userID); err != nil {
		h.logger.Error("delete task", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
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
		http.Error(w, `{"error":"month query param required"}`, http.StatusBadRequest)
		return
	}

	goals, err := h.taskSvc.ListGoalsByMonth(r.Context(), userID, month)
	if err != nil {
		h.logger.Error("list monthly goals", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goals)
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
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if body.Title == "" || body.Month == "" {
		http.Error(w, `{"error":"title and month are required"}`, http.StatusBadRequest)
		return
	}

	goal, err := h.taskSvc.CreateGoal(r.Context(), userID, body.Title, body.Month, body.Notes, body.Color)
	if err != nil {
		h.logger.Error("create monthly goal", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(goal)
}

// PATCH /goals/{id}/status
func (h *TaskHandler) UpdateGoalStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	goalID := chi.URLParam(r, "id")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	goal, err := h.taskSvc.UpdateGoalStatus(r.Context(), goalID, userID, body.Status)
	if err != nil {
		h.logger.Error("update goal status", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goal)
}

// DELETE /goals/{id}
func (h *TaskHandler) DeleteGoal(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	goalID := chi.URLParam(r, "id")

	if err := h.taskSvc.DeleteGoal(r.Context(), goalID, userID); err != nil {
		h.logger.Error("delete monthly goal", zap.Error(err))
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
