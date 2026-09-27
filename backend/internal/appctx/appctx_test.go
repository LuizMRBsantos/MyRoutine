package appctx

import (
	"context"
	"testing"
	"time"
)

func TestUserTimezoneDefaultsToSaoPauloWhenAbsent(t *testing.T) {
	loc := UserTimezone(context.Background())
	if loc.String() != "America/Sao_Paulo" {
		t.Fatalf("default timezone = %q, want %q", loc.String(), "America/Sao_Paulo")
	}
}

func TestWithTimezoneRoundTrips(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("loading Asia/Tokyo: %v", err)
	}
	ctx := WithTimezone(context.Background(), tokyo)
	if got := UserTimezone(ctx).String(); got != "Asia/Tokyo" {
		t.Fatalf("round-tripped timezone = %q, want %q", got, "Asia/Tokyo")
	}
}

func TestWithTimezoneNilFallsBackToDefault(t *testing.T) {
	ctx := WithTimezone(context.Background(), nil)
	if got := UserTimezone(ctx).String(); got != "America/Sao_Paulo" {
		t.Fatalf("timezone for nil loc = %q, want %q", got, "America/Sao_Paulo")
	}
}

func TestTodayReturnsLocalMidnight(t *testing.T) {
	sp, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatalf("loading America/Sao_Paulo: %v", err)
	}

	// 2026-03-10 02:30 UTC is still 2026-03-09 23:30 in São Paulo (UTC-3),
	// so "today" for the user must be the 9th, not the 10th.
	fixed := time.Date(2026, 3, 10, 2, 30, 0, 0, time.UTC)
	orig := nowFn
	nowFn = func() time.Time { return fixed }
	defer func() { nowFn = orig }()

	ctx := WithTimezone(context.Background(), sp)
	today := Today(ctx)

	if today.Location().String() != "America/Sao_Paulo" {
		t.Fatalf("Today location = %q, want %q", today.Location().String(), "America/Sao_Paulo")
	}
	if h, m, s := today.Clock(); h != 0 || m != 0 || s != 0 {
		t.Fatalf("Today clock = %02d:%02d:%02d, want 00:00:00", h, m, s)
	}
	if got := today.Format("2006-01-02"); got != "2026-03-09" {
		t.Fatalf("Today date = %q, want %q", got, "2026-03-09")
	}
}
