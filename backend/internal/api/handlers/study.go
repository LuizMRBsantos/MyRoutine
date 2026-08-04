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

type StudyHandler struct {
	cfg      *config.Config
	logger   *zap.Logger
	studySvc *service.StudyService
}

func NewStudyHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *StudyHandler {
	return &StudyHandler{
		cfg:      cfg,
		logger:   logger,
		studySvc: service.NewStudyService(db, logger),
	}
}

// GET /study/sessions?from&to&subject
func (h *StudyHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}

	sessions, err := h.studySvc.ListSessions(r.Context(), userID, from, to, r.URL.Query().Get("subject"))
	if err != nil {
		h.logger.Error("list study sessions", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

// POST /study/sessions
func (h *StudyHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input service.CreateStudySessionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Subject == "" {
		respondError(w, http.StatusBadRequest, "subject is required")
		return
	}
	if input.DurationMinutes <= 0 {
		respondError(w, http.StatusBadRequest, "duration_minutes must be positive")
		return
	}

	session, err := h.studySvc.CreateSession(r.Context(), userID, input)
	if err != nil {
		h.logger.Error("create study session", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	respondJSON(w, http.StatusCreated, session)
}

// DELETE /study/sessions/{id}
func (h *StudyHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	sessionID := chi.URLParam(r, "id")

	if err := h.studySvc.DeleteSession(r.Context(), sessionID, userID); err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "session not found")
			return
		}
		h.logger.Error("delete study session", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to delete session")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /study/summary
func (h *StudyHandler) Summary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	summary, err := h.studySvc.GetSummary(r.Context(), userID)
	if err != nil {
		h.logger.Error("study summary", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to build summary")
		return
	}

	respondJSON(w, http.StatusOK, summary)
}
