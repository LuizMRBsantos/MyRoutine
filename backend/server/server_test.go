package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Without a usable configuration the API must answer 503 with no internal
// details — and keep trying on later requests instead of caching the failure.
func TestServeHTTPAnswers503WhenTheAPICannotStart(t *testing.T) {
	t.Setenv("APP_ENV", "production") // skip .env loading
	t.Setenv("JWT_SECRET", "")        // config.Load fails

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("attempt %d: status = %d, want 503", i+1, rr.Code)
		}
		body := rr.Body.String()
		if !strings.Contains(body, "service unavailable") || strings.Contains(body, "JWT_SECRET") {
			t.Fatalf("attempt %d: body = %s; want a generic message without internals", i+1, body)
		}
	}
	if handler != nil {
		t.Fatal("a failed build must not be cached")
	}
}
