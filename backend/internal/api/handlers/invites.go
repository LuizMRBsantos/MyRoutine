package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/service"
)

// InviteHandler serves invite management (admin only) and the public invite
// lookup used by the registration page.
type InviteHandler struct {
	logger    *zap.Logger
	inviteSvc *service.InviteService
}

func NewInviteHandler(db *pgxpool.Pool, logger *zap.Logger) *InviteHandler {
	return &InviteHandler{logger: logger, inviteSvc: service.NewInviteService(db)}
}

// POST /admin/invites  {"email": "..."}
// The response carries the raw token once; the frontend turns it into a link.
func (h *InviteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	inv, err := h.inviteSvc.Create(r.Context(), middleware.GetUserID(r.Context()), body.Email)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidEmail):
			respondError(w, http.StatusBadRequest, "invalid email")
		case errors.Is(err, service.ErrEmailAlreadyExists):
			respondError(w, http.StatusConflict, "email already registered")
		default:
			h.logger.Error("create invite", zap.Error(err))
			respondError(w, http.StatusInternalServerError, "failed to create invite")
		}
		return
	}

	respondJSON(w, http.StatusCreated, inv)
}

// GET /admin/invites
func (h *InviteHandler) List(w http.ResponseWriter, r *http.Request) {
	invites, err := h.inviteSvc.List(r.Context())
	if err != nil {
		h.logger.Error("list invites", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list invites")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"invites": invites})
}

// DELETE /admin/invites/{id}
func (h *InviteHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := h.inviteSvc.Revoke(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "id")); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "invite not found")
			return
		}
		h.logger.Error("revoke invite", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to revoke invite")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /auth/invites/{token} — public. Tells the registration page which email
// the invite is for. Unusable invites (used, revoked, expired, unknown) all
// answer 404, so the endpoint reveals nothing beyond "this link works".
func (h *InviteHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	email, err := h.inviteSvc.Lookup(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		if errors.Is(err, service.ErrInviteInvalid) {
			respondError(w, http.StatusNotFound, "invite is invalid or expired")
			return
		}
		h.logger.Error("lookup invite", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to check invite")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"email": email})
}
