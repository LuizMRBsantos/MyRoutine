package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/myroutine/backend/internal/appctx"
)

// fakeLimiter answers with fixed values and records the keys it saw.
type fakeLimiter struct {
	allowed bool
	retry   time.Duration
	err     error
	keys    []string
}

func (f *fakeLimiter) Allow(_ context.Context, key string) (bool, time.Duration, error) {
	f.keys = append(f.keys, key)
	return f.allowed, f.retry, f.err
}

func serve(t *testing.T, l Limiter, logger *zap.Logger, ip string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	called := false
	h := RateLimit(l, logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req = req.WithContext(appctx.WithRequestMeta(req.Context(), appctx.RequestMeta{IP: ip}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr, called
}

func TestRateLimitPassesAllowedRequestsKeyedByIP(t *testing.T) {
	l := &fakeLimiter{allowed: true}
	rr, called := serve(t, l, zap.NewNop(), "203.0.113.1")
	if !called || rr.Code != http.StatusOK {
		t.Fatalf("allowed request: status %d, next called %v", rr.Code, called)
	}
	if len(l.keys) != 1 || l.keys[0] != "203.0.113.1" {
		t.Fatalf("limiter keys = %v, want the client IP", l.keys)
	}
}

func TestRateLimitBlocksWith429AndRetryAfter(t *testing.T) {
	rr, called := serve(t, &fakeLimiter{allowed: false, retry: 2500 * time.Millisecond}, zap.NewNop(), "203.0.113.1")
	if called || rr.Code != http.StatusTooManyRequests || rr.Header().Get("Retry-After") != "3" {
		t.Fatalf("status %d, Retry-After %q, next called %v; want 429, 3, false",
			rr.Code, rr.Header().Get("Retry-After"), called)
	}
}

func TestRateLimitFailsOpenAndLogsWhenTheCounterIsDown(t *testing.T) {
	core, observed := observer.New(zap.ErrorLevel)
	rr, called := serve(t, &fakeLimiter{err: errors.New("db down")}, zap.New(core), "203.0.113.1")
	if !called || rr.Code != http.StatusOK {
		t.Fatalf("limiter error: status %d, next called %v; want the request to go through", rr.Code, called)
	}
	if observed.Len() != 1 {
		t.Fatalf("logged %d errors, want 1", observed.Len())
	}
}

func TestRateLimitUsesSharedBucketWithoutIP(t *testing.T) {
	l := &fakeLimiter{allowed: true}
	serve(t, l, zap.NewNop(), "")
	if len(l.keys) != 1 || l.keys[0] != "unknown" {
		t.Fatalf("keys = %v, want the shared 'unknown' bucket", l.keys)
	}
}
