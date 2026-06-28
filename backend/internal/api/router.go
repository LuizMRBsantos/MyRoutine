package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/handlers"
	custommiddleware "github.com/myroutine/backend/internal/api/middleware"
	"github.com/myroutine/backend/internal/config"
)

// NewRouter creates and configures the Chi router with all middleware and routes.
func NewRouter(cfg *config.Config, db *pgxpool.Pool, logger *zap.Logger) http.Handler {
	r := chi.NewRouter()

	// ─── Global Middleware ──────────────────────────────────────
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommiddleware.Logger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// ─── Security Headers ───────────────────────────────────────
	r.Use(custommiddleware.SecurityHeaders())

	// ─── CORS ───────────────────────────────────────────────────
	origins := strings.Split(cfg.CORSAllowedOrigins, ",")
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ─── Handlers ───────────────────────────────────────────────
	healthHandler := handlers.NewHealthHandler(db)
	authHandler := handlers.NewAuthHandler(cfg, db, logger)
	habitHandler := handlers.NewHabitHandler(cfg, db, logger)

	// ─── Routes ─────────────────────────────────────────────────

	// Health check — sem autenticação
	r.Get("/health", healthHandler.Check)

	// Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(r chi.Router) {

		// Auth — sem JWT
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		// Protected routes — requer JWT válido
		r.Group(func(r chi.Router) {
			r.Use(custommiddleware.JWTAuth(cfg))

			// Habits
			r.Route("/habits", func(r chi.Router) {
				r.Get("/", habitHandler.List)
				r.Post("/", habitHandler.Create)
				r.Get("/stats", habitHandler.Stats)
				r.Get("/heatmap", habitHandler.Heatmap)

				r.Route("/{habitID}", func(r chi.Router) {
					r.Get("/", habitHandler.GetByID)
					r.Put("/", habitHandler.Update)
					r.Delete("/", habitHandler.Delete)
					r.Post("/checkin", habitHandler.CheckIn)
					r.Delete("/checkin", habitHandler.UndoCheckIn)
					r.Get("/logs", habitHandler.Logs)
				})
			})
		})
	})

	return r
}
