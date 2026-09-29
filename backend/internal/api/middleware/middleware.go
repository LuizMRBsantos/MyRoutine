package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/appctx"
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

// ─── Active User Middleware ───────────────────────────────────────────────────

const fallbackTimezone = "America/Sao_Paulo"

// userQuerier is the minimal slice of *pgxpool.Pool the middleware needs.
// Keeping it as an interface lets the handler be unit-tested with a fake row
// while RequireActiveUser still takes the concrete pool at the call site.
type userQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// RequireActiveUser must run after JWTAuth on the protected group. It looks up
// the authenticated user, rejects the request with 403 when the account is
// missing or inactive, and otherwise injects the user's timezone into the
// context (via appctx) for downstream "today" calculations.
func RequireActiveUser(db *pgxpool.Pool) func(http.Handler) http.Handler {
	return requireActiveUser(db)
}

func requireActiveUser(db userQuerier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			userID := GetUserID(ctx)

			var isActive, isAdmin bool
			var timezone string
			err := db.QueryRow(ctx,
				`SELECT is_active, is_admin, timezone FROM users WHERE id = $1`,
				userID,
			).Scan(&isActive, &isAdmin, &timezone)

			if err != nil || !isActive {
				http.Error(w, `{"error":"account is not active"}`, http.StatusForbidden)
				return
			}

			loc, err := time.LoadLocation(timezone)
			if err != nil || timezone == "" {
				loc, _ = time.LoadLocation(fallbackTimezone)
			}

			ctx = appctx.WithTimezone(ctx, loc)
			ctx = appctx.WithAdmin(ctx, isAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin must run after RequireActiveUser. It rejects non-admins with
// 403 so admin-only routes (invites) are invisible to regular users.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !appctx.IsAdmin(r.Context()) {
				http.Error(w, `{"error":"admin only"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequestMeta records the client IP and user agent in the context (via
// appctx) so services can write them to the audit log. It must run after
// ClientIPFromHeader: the IP comes from the trusted X-Real-IP set by nginx,
// falling back to the TCP peer for direct requests. Anything that is not a
// valid IP is dropped rather than stored.
func RequestMeta() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := middleware.GetClientIP(r.Context())
			if ip == "" {
				if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
					ip = host
				} else {
					ip = r.RemoteAddr
				}
			}
			if net.ParseIP(ip) == nil {
				ip = ""
			}

			ua := r.UserAgent()
			if len(ua) > 512 {
				ua = ua[:512]
			}

			ctx := appctx.WithRequestMeta(r.Context(), appctx.RequestMeta{IP: ip, UserAgent: ua})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
