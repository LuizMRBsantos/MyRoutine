package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/myroutine/backend/internal/appctx"
)

func captureMeta(t *testing.T, req *http.Request) appctx.RequestMeta {
	t.Helper()
	var got appctx.RequestMeta
	h := middleware.ClientIPFromHeader("X-Real-IP")(RequestMeta()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = appctx.RequestMetaFrom(r.Context())
	})))
	h.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestRequestMetaUsesTrustedRealIPAndUserAgent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:5555"
	req.Header.Set("X-Real-IP", "203.0.113.7")
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	req.Header.Set("User-Agent", "Safari/17")

	got := captureMeta(t, req)
	if got.IP != "203.0.113.7" || got.UserAgent != "Safari/17" {
		t.Fatalf("meta = %+v, want IP 203.0.113.7 and UA Safari/17", got)
	}
}

func TestRequestMetaFallsBackToPeerAddress(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:4444"

	if got := captureMeta(t, req); got.IP != "192.0.2.10" {
		t.Fatalf("IP = %q, want the peer host 192.0.2.10", got.IP)
	}
}

func TestRequestMetaDropsUnparsableIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-an-address"

	if got := captureMeta(t, req); got.IP != "" {
		t.Fatalf("IP = %q, want empty for an unparsable address", got.IP)
	}
}
