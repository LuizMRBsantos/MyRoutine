package middleware

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/appctx"
)

// Limiter counts one attempt for key and says whether it is allowed. The
// production one lives in Postgres (internal/ratelimit), shared by every
// copy of the API.
type Limiter interface {
	Allow(ctx context.Context, key string) (allowed bool, retryAfter time.Duration, err error)
}

// RateLimit caps requests per client IP on the auth routes (password
// guessing). Over the limit: 429 with Retry-After.
//
// If the limiter itself fails (database hiccup), the request goes through
// and the error is logged: a broken counter must not lock everyone out —
// wrong passwords are still wrong; the limit is an extra layer.
func RateLimit(l Limiter, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := appctx.RequestMetaFrom(r.Context()).IP
			if ip == "" {
				ip = "unknown" // no IP at all: one shared bucket, still limited
			}

			allowed, retryAfter, err := l.Allow(r.Context(), ip)
			if err != nil {
				logger.Error("rate limiter unavailable; allowing request", zap.Error(err))
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(math.Max(1, math.Ceil(retryAfter.Seconds())))))
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
