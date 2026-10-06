// Package server builds the whole MyRoutine API as one http.Handler, for
// hosts that call a handler instead of running cmd/api (Vercel Functions).
// It is the public entry point to the internal packages: code outside this
// module (the repo-root api/index.go) cannot import internal/... directly.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api"
	"github.com/myroutine/backend/internal/buildinfo"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/db"
	"github.com/myroutine/backend/internal/service"
)

var (
	mu      sync.Mutex
	handler http.Handler
)

// Handler returns the API, built once per process (one serverless instance)
// and reused across requests. A failed build is not cached: the next request
// tries again, so a transient database outage does not poison the instance.
func Handler() (http.Handler, error) {
	mu.Lock()
	defer mu.Unlock()
	if handler != nil {
		return handler, nil
	}
	h, err := build()
	if err != nil {
		return nil, err
	}
	handler = h
	return handler, nil
}

func build() (http.Handler, error) {
	newLogger := zap.NewDevelopment
	if os.Getenv("APP_ENV") == "production" {
		newLogger = zap.NewProduction
	}
	logger, err := newLogger()
	if err != nil {
		return nil, fmt.Errorf("logger: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	if cfg.RunMigrations {
		if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
			return nil, fmt.Errorf("migrations: %w", err)
		}
	}

	pool, err := db.Connect(cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	if err := service.PromoteAdmins(context.Background(), pool, cfg.AdminEmails); err != nil {
		pool.Close()
		return nil, fmt.Errorf("admins: %w", err)
	}

	logger.Info("MyRoutine API ready", zap.String("env", cfg.AppEnv), zap.String("version", buildinfo.Version))
	return api.NewRouter(cfg, pool, logger), nil
}

// ServeHTTP serves a request through the API. If the API cannot be built,
// it answers 503 without details (they go to the function log) and the next
// request retries.
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, err := Handler()
	if err != nil {
		fmt.Fprintf(os.Stderr, "myroutine: API unavailable: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"service unavailable","version":%q}`, buildinfo.Version)
		return
	}
	h.ServeHTTP(w, r)
}
