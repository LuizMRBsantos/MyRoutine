package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/service"
)

// UserHandler exposes the authenticated user's profile (/me).
type UserHandler struct {
	cfg     *config.Config
	logger  *zap.Logger
	userSvc *service.UserService
}

func NewUserHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		cfg:     cfg,
		logger:  logger,
		userSvc: service.NewUserService(db, logger),
	}
}

// GET /me
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	profile, err := h.userSvc.GetProfile(r.Context(), userID)
	if err != nil {
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("get profile failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to get profile")
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

// PATCH /me
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input service.UpdateProfileInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name != nil && *input.Name == "" {
		respondError(w, http.StatusBadRequest, "name cannot be empty")
		return
	}

	profile, err := h.userSvc.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidTimezone) {
			respondError(w, http.StatusBadRequest, "invalid timezone")
			return
		}
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("update profile failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	respondJSON(w, http.StatusOK, profile)
}

// PUT /me/password
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.NewPassword) < 8 {
		respondError(w, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}

	err := h.userSvc.ChangePassword(r.Context(), userID, body.CurrentPassword, body.NewPassword)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			// 403, not 401: for the frontend a 401 means "session expired" and
			// triggers a session refresh; this is a refused action instead.
			respondError(w, http.StatusForbidden, "current password is incorrect")
		case service.ErrNotFound:
			respondError(w, http.StatusNotFound, "user not found")
		default:
			h.logger.Error("change password failed", zap.Error(err))
			respondError(w, http.StatusInternalServerError, "failed to change password")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

// GET /me/export — LGPD: every piece of the user's data as a JSON download.
func (h *UserHandler) Export(w http.ResponseWriter, r *http.Request) {
	export, err := h.userSvc.Export(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("export account", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to export data")
		return
	}

	filename := "myroutine-dados-" + time.Now().Format("2006-01-02") + ".json"
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusOK, export)
}

// DELETE /me  {"password": "..."} — LGPD: erase the account for real.
func (h *UserHandler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.userSvc.DeleteAccount(r.Context(), middleware.GetUserID(r.Context()), body.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			respondError(w, http.StatusForbidden, "password is incorrect")
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "user not found")
		default:
			h.logger.Error("delete account", zap.Error(err))
			respondError(w, http.StatusInternalServerError, "failed to delete account")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
