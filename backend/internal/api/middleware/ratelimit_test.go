package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/myroutine/backend/internal/appctx"
)

// fakeClock lets the test move time without sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func hit(h http.Handler, ip string) int {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req = req.WithContext(appctx.WithRequestMeta(req.Context(), appctx.RequestMeta{IP: ip}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code
}

func TestRateLimitBlocksAfterLimitPerIP(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)}
	limiter := newIPLimiter(10, clock.now)
	h := limiter.middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	for i := 1; i <= 10; i++ {
		if code := hit(h, "203.0.113.1"); code != http.StatusOK {
			t.Fatalf("attempt %d = %d, want 200 (within the limit)", i, code)
		}
	}
	if code := hit(h, "203.0.113.1"); code != http.StatusTooManyRequests {
		t.Fatalf("11th attempt = %d, want 429", code)
	}
	// Another IP has its own bucket.
	if code := hit(h, "203.0.113.2"); code != http.StatusOK {
		t.Fatalf("other IP = %d, want 200", code)
	}
	// Tokens refill over time: after a minute the first IP gets in again.
	clock.t = clock.t.Add(time.Minute)
	if code := hit(h, "203.0.113.1"); code != http.StatusOK {
		t.Fatalf("after a minute = %d, want 200", code)
	}
}

func TestRateLimitSends429WithRetryAfter(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	h := newIPLimiter(1, clock.now).middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	hit(h, "198.51.100.1")

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req = req.WithContext(appctx.WithRequestMeta(req.Context(), appctx.RequestMeta{IP: "198.51.100.1"}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests || rr.Header().Get("Retry-After") != "60" {
		t.Fatalf("status %d, Retry-After %q; want 429 and 60", rr.Code, rr.Header().Get("Retry-After"))
	}
}

func TestRateLimitForgetsIdleIPs(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	limiter := newIPLimiter(10, clock.now)
	h := limiter.middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	hit(h, "192.0.2.1")

	clock.t = clock.t.Add(11 * time.Minute)
	hit(h, "192.0.2.2") // triggers the sweep

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if _, ok := limiter.entries["192.0.2.1"]; ok {
		t.Fatal("an IP idle for over 10 minutes must be forgotten")
	}
}
