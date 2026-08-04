package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	cfg     *config.Config
	db      *pgxpool.Pool
	logger  *zap.Logger
	authSvc *service.AuthService
}

func NewAuthHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		cfg:     cfg,
		db:      db,
		logger:  logger,
		authSvc: service.NewAuthService(cfg, db, logger),
	}
}

// ─── Request / Response types ─────────────────────────────────────────────────

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Timezone string `json:"timezone"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         userDTO   `json:"user"`
}

type userDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL *string   `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// Register creates a new user account.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "name, email and password are required")
		return
	}

	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	if req.Timezone == "" {
		req.Timezone = "America/Sao_Paulo"
	}

	result, err := h.authSvc.Register(r.Context(), req.Name, req.Email, req.Password, req.Timezone)
	if err != nil {
		if err == service.ErrEmailAlreadyExists {
			respondError(w, http.StatusConflict, "email already registered")
			return
		}
		h.logger.Error("register failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// Login authenticates a user and returns JWT tokens.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.authSvc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			// Security: never reveal if email exists or password is wrong
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		h.logger.Error("login failed", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Refresh issues a new access token using a valid refresh token.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.authSvc.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Logout revokes the refresh token.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.authSvc.Logout(r.Context(), body.RefreshToken); err != nil {
		h.logger.Error("logout failed", zap.Error(err))
	}

	// Always return 200 — don't reveal token state
	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

