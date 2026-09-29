package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/myroutine/backend/internal/appctx"
)

// RateLimit caps requests per client IP (token bucket: `perMinute` tokens
// refilled over a minute, bursts up to `perMinute`). It guards the auth
// routes against password guessing now that no nginx sits in front in AWS.
// Over the limit: 429 with Retry-After. Idle IPs are forgotten after 10 min.
func RateLimit(perMinute int) func(http.Handler) http.Handler {
	return newIPLimiter(perMinute, time.Now).middleware
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipLimiter struct {
	mu        sync.Mutex
	entries   map[string]*limiterEntry
	every     rate.Limit
	burst     int
	retry     string
	now       func() time.Time
	lastSweep time.Time
}

const limiterIdleTTL = 10 * time.Minute

func newIPLimiter(perMinute int, now func() time.Time) *ipLimiter {
	return &ipLimiter{
		entries:   map[string]*limiterEntry{},
		every:     rate.Limit(float64(perMinute) / 60),
		burst:     perMinute,
		retry:     strconv.Itoa(int(math.Ceil(60 / float64(perMinute)))),
		now:       now,
		lastSweep: now(),
	}
}

func (l *ipLimiter) allow(ip string) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) > limiterIdleTTL {
		for k, e := range l.entries {
			if now.Sub(e.lastSeen) > limiterIdleTTL {
				delete(l.entries, k)
			}
		}
		l.lastSweep = now
	}

	e, ok := l.entries[ip]
	if !ok {
		e = &limiterEntry{limiter: rate.NewLimiter(l.every, l.burst)}
		l.entries[ip] = e
	}
	e.lastSeen = now
	return e.limiter.AllowN(now, 1)
}

func (l *ipLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := appctx.RequestMetaFrom(r.Context()).IP
		if ip == "" {
			ip = "unknown" // no IP at all: one shared bucket, still limited
		}
		if !l.allow(ip) {
			w.Header().Set("Retry-After", l.retry)
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
