package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/myroutine/backend/internal/buildinfo"
)

// HealthHandler handles the health check endpoint.
type HealthHandler struct {
	db *pgxpool.Pool
}

func NewHealthHandler(db *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{db: db}
}

type healthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
}

// Check returns a 200 if all services are healthy, 503 otherwise.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	services := map[string]string{}
	overallStatus := "healthy"

	// Check PostgreSQL
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		services["postgres"] = "unhealthy"
		overallStatus = "degraded"
	} else {
		services["postgres"] = "healthy"
	}

	resp := healthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Version:   buildinfo.Version,
		Services:  services,
	}

	statusCode := http.StatusOK
	if overallStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}
