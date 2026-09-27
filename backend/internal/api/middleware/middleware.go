package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/config"
)

// ─── Logger Middleware ────────────────────────────────────────────────────────

// Logger creates a structured request logger middleware using zap.
// Logs: method, path, status, latency, request ID, IP.
func Logger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			ip := middleware.GetClientIP(r.Context())
			if ip == "" {
				ip = r.RemoteAddr
			}

			defer func() {
				logger.Info("request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Duration("latency", time.Since(start)),
					zap.String("request_id", middleware.GetReqID(r.Context())),
					zap.String("ip", ip),
					zap.String("user_agent", r.UserAgent()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// ─── Security Headers Middleware ─────────────────────────────────────────────

// SecurityHeaders adds security HTTP headers to every response.
// These are the same headers recommended by OWASP.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			// Note: HSTS only in production (requires HTTPS)
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─── JWT Auth Middleware ──────────────────────────────────────────────────────

type contextKey string

const UserIDKey contextKey = "user_id"

// JWTAuth validates the Bearer token in Authorization header.
// On success, injects user_id into request context.
func JWTAuth(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"authorization header required"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"invalid token claims"}`, http.StatusUnauthorized)
				return
			}

			userID, ok := claims["sub"].(string)
			if !ok {
				http.Error(w, `{"error":"invalid token subject"}`, http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = setContextValue(ctx, UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
