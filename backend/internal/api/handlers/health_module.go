package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/appctx"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/service"
)

// HealthModuleHandler exposes the Health module (training volume, body
// metrics). Named to avoid clashing with the infrastructure health check.
type HealthModuleHandler struct {
	cfg       *config.Config
	logger    *zap.Logger
	healthSvc *service.HealthService
}

func NewHealthModuleHandler(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) *HealthModuleHandler {
	return &HealthModuleHandler{
		cfg:       cfg,
		logger:    logger,
		healthSvc: service.NewHealthService(db, logger),
	}
}

// GET /health-module/activities?from&to
func (h *HealthModuleHandler) ListActivities(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	today := appctx.Today(r.Context())
	if to == "" {
		to = today.Format("2006-01-02")
	}
	if from == "" {
		from = today.AddDate(0, -1, 0).Format("2006-01-02")
	}

	activities, err := h.healthSvc.ListActivities(r.Context(), userID, from, to)
	if err != nil {
		h.logger.Error("list activities", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list activities")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"activities": activities})
}

// GET /health-module/summary?weeks=4
func (h *HealthModuleHandler) Summary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	weeks, _ := strconv.Atoi(r.URL.Query().Get("weeks"))
	summary, err := h.healthSvc.GetSummary(r.Context(), userID, weeks)
	if err != nil {
		h.logger.Error("health summary", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to build summary")
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

// GET /health-module/body-metrics
func (h *HealthModuleHandler) ListBodyMetrics(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	metrics, err := h.healthSvc.ListBodyMetrics(r.Context(), userID, limit)
	if err != nil {
		h.logger.Error("list body metrics", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list body metrics")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"body_metrics": metrics})
}

// POST /health-module/body-metrics — upsert por dia
func (h *HealthModuleHandler) UpsertBodyMetric(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var body struct {
		MeasuredOn string   `json:"measured_on"`
		WeightKg   *float64 `json:"weight_kg"`
		Notes      *string  `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.WeightKg == nil && body.Notes == nil {
		respondError(w, http.StatusBadRequest, "weight_kg or notes required")
		return
	}
	if body.WeightKg != nil && (*body.WeightKg <= 0 || *body.WeightKg > 500) {
		respondError(w, http.StatusBadRequest, "weight_kg out of range")
		return
	}

	metric, err := h.healthSvc.UpsertBodyMetric(r.Context(), userID, body.MeasuredOn, body.WeightKg, body.Notes)
	if err != nil {
		h.logger.Error("upsert body metric", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to save body metric")
		return
	}

	respondJSON(w, http.StatusOK, metric)
}
