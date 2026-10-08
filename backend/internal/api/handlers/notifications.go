package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/service"
)

// NotificationHandler serves the notification settings and this device's
// push subscription (/notifications/*).
type NotificationHandler struct {
	cfg    *config.Config
	logger *zap.Logger
	svc    *service.NotificationService
}

func NewNotificationHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *NotificationHandler {
	return &NotificationHandler{cfg: cfg, logger: logger, svc: service.NewNotificationService(db)}
}

// GET /notifications/config — whether push is configured, and the public
// VAPID key the browser needs to subscribe.
func (h *NotificationHandler) Config(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]any{"enabled": h.cfg.NotificationsEnabled()}
	if h.cfg.NotificationsEnabled() {
		resp["vapid_public_key"] = h.cfg.VAPIDPublicKey
	}
	respondJSON(w, http.StatusOK, resp)
}

// GET /notifications/settings
func (h *NotificationHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	view, err := h.svc.Get(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		h.logger.Error("get notification settings", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to load notification settings")
		return
	}
	respondJSON(w, http.StatusOK, view)
}

// PUT /notifications/settings
func (h *NotificationHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var in service.NotificationSettings
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.svc.Update(r.Context(), middleware.GetUserID(r.Context()), in)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSettings) {
			respondError(w, http.StatusBadRequest, "invalid notification settings")
			return
		}
		h.logger.Error("update notification settings", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to save notification settings")
		return
	}
	respondJSON(w, http.StatusOK, view)
}

// POST /notifications/subscriptions — the browser's PushSubscription.toJSON().
func (h *NotificationHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.NotificationsEnabled() {
		respondError(w, http.StatusServiceUnavailable, "notifications are not configured")
		return
	}
	var in service.PushSubscriptionInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.Subscribe(r.Context(), middleware.GetUserID(r.Context()), in, r.UserAgent()); err != nil {
		if errors.Is(err, service.ErrInvalidSubscription) {
			respondError(w, http.StatusBadRequest, "invalid push subscription")
			return
		}
		h.logger.Error("push subscribe", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to subscribe")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /notifications/subscriptions  {"endpoint": "..."}
func (h *NotificationHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body); err != nil || body.Endpoint == "" {
		respondError(w, http.StatusBadRequest, "endpoint is required")
		return
	}
	if err := h.svc.Unsubscribe(r.Context(), middleware.GetUserID(r.Context()), body.Endpoint); err != nil {
		h.logger.Error("push unsubscribe", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to unsubscribe")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
