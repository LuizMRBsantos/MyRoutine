package push

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/myroutine/backend/internal/service"
)

// A real-format device key pair, generated per test run.
func testTarget(t *testing.T, endpoint string) service.PushTarget {
	t.Helper()
	priv, pub, err := webpush.GenerateVAPIDKeys() // same P-256 format as a browser's p256dh
	if err != nil || priv == "" {
		t.Fatal(err)
	}
	return service.PushTarget{Endpoint: endpoint, P256dh: pub, Auth: "AAAAAAAAAAAAAAAAAAAAAA"}
}

func TestSendInterpretsPushServiceAnswers(t *testing.T) {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		status  int
		gone    bool
		wantErr bool
	}{
		{http.StatusCreated, false, false},
		{http.StatusGone, true, false},
		{http.StatusNotFound, true, false},
		{http.StatusTooManyRequests, false, true},
	} {
		var gotAuth, gotEncoding string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth, gotEncoding = r.Header.Get("Authorization"), r.Header.Get("Content-Encoding")
			w.WriteHeader(tc.status)
		}))
		sender := NewWebPush(pub, priv, "mailto:owner@example.com")
		gone, err := sender.Send(context.Background(), testTarget(t, srv.URL), []byte(`{"title":"x"}`))
		srv.Close()

		if gone != tc.gone || (err != nil) != tc.wantErr {
			t.Errorf("status %d: gone=%v err=%v, want gone=%v err?=%v", tc.status, gone, err, tc.gone, tc.wantErr)
		}
		if gotEncoding != "aes128gcm" || len(gotAuth) < 20 {
			t.Errorf("status %d: request not encrypted/signed (encoding %q, auth %q)", tc.status, gotEncoding, gotAuth)
		}
	}
}

func TestSendDoesNotFollowRedirects(t *testing.T) {
	priv, pub, _ := webpush.GenerateVAPIDKeys()
	hit := false
	elsewhere := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hit = true }))
	defer elsewhere.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL, http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	_, err := NewWebPush(pub, priv, "owner@example.com").Send(context.Background(), testTarget(t, srv.URL), []byte(`{}`))
	if err == nil || hit {
		t.Fatalf("redirect: err=%v, followed=%v; want an error and no request elsewhere", err, hit)
	}
}
