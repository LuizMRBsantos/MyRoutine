// Package appctx holds request-scoped values that cross package boundaries
// (timezone today, more later) together with the typed context keys used to
// carry them. It deliberately imports nothing from internal/api/middleware:
// the middleware depends on appctx, never the other way around, so the key
// definitions live here to keep that dependency acyclic.
package appctx

import (
	"context"
	"time"
)

// contextKey is unexported so no other package can collide with these keys.
type contextKey string

const (
	timezoneKey contextKey = "timezone"
	adminKey    contextKey = "is_admin"
)

// defaultTimezone is the app-wide fallback matching the users.timezone column
// default. Loaded once at init; the binary embeds the tzdata database (see the
// blank time/tzdata import in cmd/api), so this succeeds even on a scratch image.
var defaultTimezone = mustLoadLocation("America/Sao_Paulo")

// nowFn is the clock source, overridable in tests.
var nowFn = time.Now

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Only reachable if tzdata is missing from the build; a hard failure
		// here is preferable to silently serving UTC everywhere.
		panic("appctx: cannot load default timezone " + name + ": " + err.Error())
	}
	return loc
}

// WithTimezone stores the user's timezone in the context. A nil location is
// replaced with the default so callers of UserTimezone always get a usable value.
func WithTimezone(ctx context.Context, loc *time.Location) context.Context {
	if loc == nil {
		loc = defaultTimezone
	}
	return context.WithValue(ctx, timezoneKey, loc)
}

// UserTimezone returns the timezone stored by WithTimezone, or the default
// (America/Sao_Paulo) when none is present.
func UserTimezone(ctx context.Context) *time.Location {
	if loc, ok := ctx.Value(timezoneKey).(*time.Location); ok && loc != nil {
		return loc
	}
	return defaultTimezone
}

// Today returns midnight of the current day in the user's timezone.
//
// The returned time is anchored in the user's location, so consumers typically
// use it as today.Format("2006-01-02") for a calendar-day string, or pass it to
// a query as $n::date. It is NOT a UTC instant — do not compare it against a
// server-clock UTC time without converting.
func Today(ctx context.Context) time.Time {
	loc := UserTimezone(ctx)
	now := nowFn().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
}

// WithAdmin records whether the authenticated user is an admin.
func WithAdmin(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, adminKey, isAdmin)
}

// IsAdmin reports whether the authenticated user is an admin. False when
// unknown, so a missing value never grants admin access.
func IsAdmin(ctx context.Context) bool {
	v, _ := ctx.Value(adminKey).(bool)
	return v
}
