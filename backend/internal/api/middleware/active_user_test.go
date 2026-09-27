package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/myroutine/backend/internal/appctx"
)

// fakeRow implements pgx.Row for the single-row lookup the middleware performs.
type fakeRow struct {
	isActive bool
	timezone string
	err      error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	// SELECT is_active, timezone FROM users WHERE id = $1
	if len(dest) != 2 {
		panic("unexpected scan arity")
	}
	*dest[0].(*bool) = r.isActive
	*dest[1].(*string) = r.timezone
	return nil
}

// fakeQuerier implements the minimal querier surface the middleware needs.
type fakeQuerier struct {
	row fakeRow
}

func (q fakeQuerier) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return q.row
}

func newAuthedRequest(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	ctx := setContextValue(req.Context(), UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestRequireActiveUserRejectsInactiveAccount(t *testing.T) {
	q := fakeQuerier{row: fakeRow{isActive: false, timezone: "America/Sao_Paulo"}}

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })

	rr := httptest.NewRecorder()
	requireActiveUser(q)(next).ServeHTTP(rr, newAuthedRequest("11111111-1111-1111-1111-111111111111"))

	if nextCalled {
		t.Fatal("next handler was called for an inactive account, want it skipped")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding body %q: %v", rr.Body.String(), err)
	}
	if body["error"] != "account is not active" {
		t.Fatalf("body error = %q, want %q", body["error"], "account is not active")
	}
}

func TestRequireActiveUserRejectsMissingUser(t *testing.T) {
	q := fakeQuerier{row: fakeRow{err: pgx.ErrNoRows}}

	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })

	rr := httptest.NewRecorder()
	requireActiveUser(q)(next).ServeHTTP(rr, newAuthedRequest("22222222-2222-2222-2222-222222222222"))

	if nextCalled {
		t.Fatal("next handler was called for a missing user, want it skipped")
	}
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestRequireActiveUserInjectsUserTimezone(t *testing.T) {
	q := fakeQuerier{row: fakeRow{isActive: true, timezone: "America/Sao_Paulo"}}

	var gotTZ string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotTZ = appctx.UserTimezone(r.Context()).String()
	})

	rr := httptest.NewRecorder()
	requireActiveUser(q)(next).ServeHTTP(rr, newAuthedRequest("33333333-3333-3333-3333-333333333333"))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if gotTZ != "America/Sao_Paulo" {
		t.Fatalf("injected timezone = %q, want %q", gotTZ, "America/Sao_Paulo")
	}
}

func TestRequireActiveUserFallsBackOnInvalidTimezone(t *testing.T) {
	q := fakeQuerier{row: fakeRow{isActive: true, timezone: "Not/AZone"}}

	var gotTZ string
	handlerReached := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		handlerReached = true
		gotTZ = appctx.UserTimezone(r.Context()).String()
	})

	rr := httptest.NewRecorder()
	requireActiveUser(q)(next).ServeHTTP(rr, newAuthedRequest("44444444-4444-4444-4444-444444444444"))

	if !handlerReached {
		t.Fatal("next handler was not called; an invalid timezone must fall back, not drop the request")
	}
	if gotTZ != "America/Sao_Paulo" {
		t.Fatalf("fallback timezone = %q, want %q", gotTZ, "America/Sao_Paulo")
	}
}
