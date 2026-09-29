package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/service"
)

// maxJournalBody caps the request (20k characters of text fit well within).
const maxJournalBody = 256 << 10

// JournalHandler serves the Track Day journal (/journal/{date}).
type JournalHandler struct {
	logger     *zap.Logger
	journalSvc *service.JournalService
}

func NewJournalHandler(db *pgxpool.Pool, logger *zap.Logger) *JournalHandler {
	return &JournalHandler{logger: logger, journalSvc: service.NewJournalService(db, logger)}
}

// GET /journal/{date}
func (h *JournalHandler) Get(w http.ResponseWriter, r *http.Request) {
	out, err := h.journalSvc.Get(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "date"))
	h.respond(w, out, err, "load journal")
}

// PUT /journal/{date}  {"content": "..."}
func (h *JournalHandler) Save(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJournalBody)
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	out, err := h.journalSvc.Save(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "date"), body.Content)
	h.respond(w, out, err, "save journal")
}

// POST /journal/{date}/items
//
//	{"line": "...", "kind": "transaction", "expense": {...}}
//	{"line": "...", "kind": "workout", "workout": {...}}
func (h *JournalHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxJournalBody)
	var body struct {
		Line    string                        `json:"line"`
		Kind    string                        `json:"kind"`
		Expense *service.RegisterExpenseInput `json:"expense"`
		Workout *service.RegisterWorkoutInput `json:"workout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if service.JournalLineKey(body.Line) == "" {
		respondError(w, http.StatusBadRequest, "line is required")
		return
	}

	userID := middleware.GetUserID(r.Context())
	date := chi.URLParam(r, "date")

	switch body.Kind {
	case service.JournalKindExpense:
		if body.Expense == nil || body.Expense.AmountCents <= 0 || strings.TrimSpace(body.Expense.Description) == "" {
			respondError(w, http.StatusBadRequest, "expense needs a positive amount and a description")
			return
		}
		out, err := h.journalSvc.RegisterExpense(r.Context(), userID, date, body.Line, *body.Expense)
		h.respond(w, out, err, "register expense")
	case service.JournalKindWorkout:
		if body.Workout == nil || body.Workout.HabitID == "" {
			respondError(w, http.StatusBadRequest, "workout needs a habit_id")
			return
		}
		out, err := h.journalSvc.RegisterWorkout(r.Context(), userID, date, body.Line, *body.Workout)
		h.respond(w, out, err, "register workout")
	default:
		respondError(w, http.StatusBadRequest, "kind must be transaction or workout")
	}
}

// DELETE /journal/{date}/items/{sourceID}
func (h *JournalHandler) Unregister(w http.ResponseWriter, r *http.Request) {
	out, err := h.journalSvc.Unregister(r.Context(), middleware.GetUserID(r.Context()),
		chi.URLParam(r, "date"), chi.URLParam(r, "sourceID"))
	h.respond(w, out, err, "undo journal item")
}

func (h *JournalHandler) respond(w http.ResponseWriter, out *service.JournalDTO, err error, what string) {
	if err == nil {
		respondJSON(w, http.StatusOK, out)
		return
	}
	var tooBig *http.MaxBytesError
	switch {
	case errors.Is(err, service.ErrInvalidDate):
		respondError(w, http.StatusBadRequest, "invalid date")
	case errors.Is(err, service.ErrJournalTooLong), errors.As(err, &tooBig):
		respondError(w, http.StatusRequestEntityTooLarge, "journal entry too long")
	case errors.Is(err, service.ErrAlreadyLogged):
		respondError(w, http.StatusConflict, "habit already checked in that day")
	case errors.Is(err, service.ErrCheckInRejected):
		// The habit rule's own message (e.g. deadline passed) is safe to show.
		respondError(w, http.StatusUnprocessableEntity, strings.TrimPrefix(err.Error(), service.ErrCheckInRejected.Error()+": "))
	case errors.Is(err, service.ErrInvalidReference):
		respondError(w, http.StatusBadRequest, "invalid reference")
	case errors.Is(err, service.ErrNotFound):
		respondError(w, http.StatusNotFound, "not found")
	default:
		h.logger.Error(what, zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to "+what)
	}
}
