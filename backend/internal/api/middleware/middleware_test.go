package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerLogsTrustedXRealIPInsteadOfSpoofableForwardedHeaders(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	handler := RequestMeta("X-Real-IP")(Logger(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Header.Set("X-Real-IP", "203.0.113.10")
	req.Header.Set("X-Forwarded-For", "198.51.100.20")
	req.Header.Set("True-Client-IP", "198.51.100.30")

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got := observed.All()[0].ContextMap()["ip"]; got != "203.0.113.10" {
		t.Fatalf("logged IP = %q, want trusted X-Real-IP %q", got, "203.0.113.10")
	}
}

func TestLoggerFallsBackToRemoteAddrWithoutTrustedClientIP(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	handler := RequestMeta("X-Real-IP")(Logger(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:54321"

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got := observed.All()[0].ContextMap()["ip"]; got != "127.0.0.1" {
		t.Fatalf("logged IP = %q, want the peer address %q", got, "127.0.0.1")
	}
}
