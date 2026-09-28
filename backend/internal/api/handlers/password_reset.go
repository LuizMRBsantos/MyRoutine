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

// PasswordResetHandler serves admin-issued reset links: the admin creates one
// (protected), the user opens it and sets a new password (public).
type PasswordResetHandler struct {
	logger   *zap.Logger
	resetSvc *service.PasswordResetService
}

func NewPasswordResetHandler(db *pgxpool.Pool, logger *zap.Logger) *PasswordResetHandler {
	return &PasswordResetHandler{logger: logger, resetSvc: service.NewPasswordResetService(db)}
}

// POST /admin/password-resets  {"email": "..."}
func (h *PasswordResetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	reset, err := h.resetSvc.Create(r.Context(), middleware.GetUserID(r.Context()), body.Email)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "no account with this email")
			return
		}
		h.logger.Error("create password reset", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create reset link")
		return
	}
	respondJSON(w, http.StatusCreated, reset)
}

// GET /auth/password-resets/{token} — public; unusable links answer 404.
func (h *PasswordResetHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	email, err := h.resetSvc.Lookup(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		if errors.Is(err, service.ErrResetInvalid) {
			respondError(w, http.StatusNotFound, "reset link is invalid or expired")
			return
		}
		h.logger.Error("lookup password reset", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to check reset link")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"email": email})
}

// POST /auth/password-resets/{token}  {"password": "..."} — public.
func (h *PasswordResetHandler) Reset(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.resetSvc.Reset(r.Context(), chi.URLParam(r, "token"), body.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrPasswordTooShort):
			respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		case errors.Is(err, service.ErrResetInvalid):
			respondError(w, http.StatusNotFound, "reset link is invalid or expired")
		default:
			h.logger.Error("reset password", zap.Error(err))
			respondError(w, http.StatusInternalServerError, "failed to reset password")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
