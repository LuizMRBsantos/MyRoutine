// Package push sends Web Push notifications (VAPID) to subscribed devices.
package push

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/myroutine/backend/internal/service"
)

// WebPush implements service.PushSender with the VAPID keys from config.
type WebPush struct {
	publicKey  string
	privateKey string
	subject    string
	client     *http.Client
}

func NewWebPush(publicKey, privateKey, subject string) *WebPush {
	return &WebPush{
		publicKey:  publicKey,
		privateKey: privateKey,
		// The library adds "mailto:" itself; accept the subject either way.
		subject: strings.TrimPrefix(subject, "mailto:"),
		client: &http.Client{
			Timeout: 10 * time.Second,
			// Endpoints are allowlisted at subscribe time; never follow a
			// redirect somewhere else.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

var _ service.PushSender = (*WebPush)(nil)

// Send encrypts payload for the device and posts it to its push service.
func (w *WebPush) Send(ctx context.Context, t service.PushTarget, payload []byte) (bool, error) {
	resp, err := webpush.SendNotificationWithContext(ctx, payload,
		&webpush.Subscription{Endpoint: t.Endpoint, Keys: webpush.Keys{P256dh: t.P256dh, Auth: t.Auth}},
		&webpush.Options{
			HTTPClient:      w.client,
			Subscriber:      w.subject,
			VAPIDPublicKey:  w.publicKey,
			VAPIDPrivateKey: w.privateKey,
			TTL:             60 * 60, // a reminder older than an hour is useless
			Urgency:         webpush.UrgencyHigh,
		})
	if err != nil {
		return false, fmt.Errorf("sending push: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return true, nil
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return false, nil
	default:
		return false, fmt.Errorf("push service answered %d", resp.StatusCode)
	}
}
