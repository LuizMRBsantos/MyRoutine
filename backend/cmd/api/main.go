package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Embed the IANA timezone database into the binary. The production image is
	// built FROM scratch and has no /usr/share/zoneinfo, so without this
	// time.LoadLocation would fail for every per-user timezone.
	_ "time/tzdata"

	"github.com/myroutine/backend/internal/api"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/db"
	"go.uber.org/zap"
)

func main() {
	// ─── Logger ───────────────────────────────────────────────────
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync() //nolint:errcheck

	// ─── Config ───────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// ─── Database ─────────────────────────────────────────────────
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}
	logger.Info("migrations applied")

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()
	logger.Info("database connected", zap.String("host", cfg.DBHost))

	// ─── Router ───────────────────────────────────────────────────
	router := api.NewRouter(cfg, pool, logger)

	// ─── Server ───────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.AppPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ─── Graceful Shutdown ────────────────────────────────────────
	go func() {
		logger.Info("🚀 MyRoutine API starting", zap.String("port", cfg.AppPort), zap.String("env", cfg.AppEnv))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}
