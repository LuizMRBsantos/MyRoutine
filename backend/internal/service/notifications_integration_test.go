package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func pushSub(endpoint string) PushSubscriptionInput {
	var in PushSubscriptionInput
	in.Endpoint = endpoint
	in.Keys.P256dh = "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM"
	in.Keys.Auth = "tBHItJI5svbpez7KI4CCXg" // gitleaks:allow — public example push key (test fixture)
	return in
}

func uniqueEndpoint(host string) string {
	return fmt.Sprintf("https://%s/push/%d", host, time.Now().UnixNano())
}

func TestSubscribeOnlyAcceptsOfficialPushServices(t *testing.T) {
	svc := NewNotificationService(requireDB(t))
	userID := createTestUser(t)
	ctx := context.Background()

	for _, bad := range []string{
		"https://evil.example.com/push/1",               // arbitrary host (SSRF)
		"https://fcm.googleapis.com.evil.example/push",  // suffix trick
		"http://fcm.googleapis.com/fcm/send/abc",        // no TLS
		"https://fcm.googleapis.com:8443/fcm/send/abc",  // odd port
		"https://user:pw@fcm.googleapis.com/fcm/send/x", // credentials in URL
		"https://169.254.169.254/latest/meta-data",      // metadata IP
	} {
		if err := svc.Subscribe(ctx, userID, pushSub(bad), ""); !errors.Is(err, ErrInvalidSubscription) {
			t.Errorf("%s: err = %v, want ErrInvalidSubscription", bad, err)
		}
	}

	for _, good := range []string{
		uniqueEndpoint("web.push.apple.com"),
		uniqueEndpoint("fcm.googleapis.com"),
		uniqueEndpoint("updates.push.services.mozilla.com"),
		uniqueEndpoint("wns2-bl2p.notify.windows.com"),
	} {
		if err := svc.Subscribe(ctx, userID, pushSub(good), "Safari"); err != nil {
			t.Errorf("%s: err = %v, want accepted", good, err)
		}
	}

	bad := pushSub(uniqueEndpoint("fcm.googleapis.com"))
	bad.Keys.Auth = "not base64!"
	if err := svc.Subscribe(ctx, userID, bad, ""); !errors.Is(err, ErrInvalidSubscription) {
		t.Errorf("malformed keys: err = %v, want ErrInvalidSubscription", err)
	}
}

func TestFirstSubscriptionStartsWithProductDefaults(t *testing.T) {
	svc := NewNotificationService(requireDB(t))
	userID := createTestUser(t)
	ctx := context.Background()

	before, err := svc.Get(ctx, userID)
	if err != nil || before.Devices != 0 || before.Settings != DefaultNotificationSettings {
		t.Fatalf("before subscribing: %+v, err %v", before, err)
	}

	if err := svc.Subscribe(ctx, userID, pushSub(uniqueEndpoint("web.push.apple.com")), "iPhone"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	after, err := svc.Get(ctx, userID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	want := NotificationSettings{TaskReminders: true, TaskLeadMinutes: 15, MorningDigest: true, MorningTime: "07:00", EveningDigest: false, EveningTime: "21:00"}
	if after.Devices != 1 || after.Settings != want {
		t.Fatalf("after subscribing: %+v, want 1 device and %+v", after, want)
	}
}

func TestSameDeviceMovesToWhoeverLogsIn(t *testing.T) {
	svc := NewNotificationService(requireDB(t))
	ctx := context.Background()
	first, second := createTestUser(t), createTestUser(t)
	device := pushSub(uniqueEndpoint("fcm.googleapis.com"))

	if err := svc.Subscribe(ctx, first, device, ""); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := svc.Subscribe(ctx, second, device, ""); err != nil {
		t.Fatalf("second: %v", err)
	}
	a, _ := svc.Get(ctx, first)
	b, _ := svc.Get(ctx, second)
	if a.Devices != 0 || b.Devices != 1 {
		t.Fatalf("devices: first %d (want 0), second %d (want 1)", a.Devices, b.Devices)
	}
}

func TestUpdateSettingsValidatesAndSaves(t *testing.T) {
	svc := NewNotificationService(requireDB(t))
	userID := createTestUser(t)
	ctx := context.Background()

	for _, bad := range []NotificationSettings{
		{TaskLeadMinutes: 2, MorningTime: "07:00", EveningTime: "21:00"},
		{TaskLeadMinutes: 500, MorningTime: "07:00", EveningTime: "21:00"},
		{TaskLeadMinutes: 15, MorningTime: "25:00", EveningTime: "21:00"},
		{TaskLeadMinutes: 15, MorningTime: "07:00", EveningTime: "nine"},
	} {
		if _, err := svc.Update(ctx, userID, bad); !errors.Is(err, ErrInvalidSettings) {
			t.Errorf("%+v: err = %v, want ErrInvalidSettings", bad, err)
		}
	}

	want := NotificationSettings{TaskReminders: false, TaskLeadMinutes: 30, MorningDigest: true, MorningTime: "06:30", EveningDigest: true, EveningTime: "22:00"}
	got, err := svc.Update(ctx, userID, want)
	if err != nil || got.Settings != want {
		t.Fatalf("update: %+v, err %v; want %+v", got, err, want)
	}
}

func TestUnsubscribeOnlyRemovesOwnDevice(t *testing.T) {
	svc := NewNotificationService(requireDB(t))
	ctx := context.Background()
	owner, other := createTestUser(t), createTestUser(t)
	device := pushSub(uniqueEndpoint("web.push.apple.com"))
	if err := svc.Subscribe(ctx, owner, device, ""); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	if err := svc.Unsubscribe(ctx, other, device.Endpoint); err != nil {
		t.Fatalf("other unsubscribe: %v", err)
	}
	if v, _ := svc.Get(ctx, owner); v.Devices != 1 {
		t.Fatal("another user must not remove the owner's device")
	}
	if err := svc.Unsubscribe(ctx, owner, device.Endpoint); err != nil {
		t.Fatalf("owner unsubscribe: %v", err)
	}
	if v, _ := svc.Get(ctx, owner); v.Devices != 0 {
		t.Fatal("owner's device should be gone")
	}
}
