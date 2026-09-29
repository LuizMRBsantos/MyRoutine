package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/myroutine/backend/internal/appctx"
)

func captureMeta(t *testing.T, trustedHeader string, req *http.Request) appctx.RequestMeta {
	t.Helper()
	var got appctx.RequestMeta
	RequestMeta(trustedHeader)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = appctx.RequestMetaFrom(r.Context())
	})).ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestRequestMetaUsesTrustedRealIPAndUserAgent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:5555"
	req.Header.Set("X-Real-IP", "203.0.113.7")
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	req.Header.Set("User-Agent", "Safari/17")

	got := captureMeta(t, "X-Real-IP", req)
	if got.IP != "203.0.113.7" || got.UserAgent != "Safari/17" {
		t.Fatalf("meta = %+v, want IP 203.0.113.7 and UA Safari/17", got)
	}
}

func TestRequestMetaFallsBackToPeerAddress(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.10:4444"

	if got := captureMeta(t, "X-Real-IP", req); got.IP != "192.0.2.10" {
		t.Fatalf("IP = %q, want the peer host 192.0.2.10", got.IP)
	}
}

func TestRequestMetaDropsUnparsableIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-an-address"
	req.Header.Set("X-Real-IP", "<script>")

	if got := captureMeta(t, "X-Real-IP", req); got.IP != "" {
		t.Fatalf("IP = %q, want empty for unparsable values", got.IP)
	}
}

// In AWS the viewer IP comes from CloudFront-Viewer-Address, "ip:port" —
// for IPv6 without brackets. A client-sent X-Real-IP must not count there.
func TestRequestMetaReadsCloudFrontViewerAddress(t *testing.T) {
	cases := map[string]string{
		"198.51.100.10:46532":            "198.51.100.10",
		"2001:db8:85a3::8a2e:7334:46532": "2001:db8:85a3::8a2e:7334",
		"[2001:db8::1]:443":              "2001:db8::1",
	}
	for header, want := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.5:1234"
		req.Header.Set("CloudFront-Viewer-Address", header)
		req.Header.Set("X-Real-IP", "6.6.6.6") // spoofed by the client

		if got := captureMeta(t, "CloudFront-Viewer-Address", req); got.IP != want {
			t.Errorf("CloudFront-Viewer-Address %q → IP %q, want %q", header, got.IP, want)
		}
	}
}
