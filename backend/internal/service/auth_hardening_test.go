package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/myroutine/backend/internal/config"
)

const testJWTSecret = "test-secret-that-is-at-least-32-characters-long" // gitleaks:allow — fake secret, tests only

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()
	pool := requireDB(t)
	cfg := &config.Config{
		JWTSecret:            testJWTSecret,
		JWTExpiryHours:       "24",
		JWTRefreshExpiryDays: "30",
	}
	return NewAuthService(cfg, pool, testLogger)
}

// uniqueEmail returns a mixed-case, per-test email so tests never collide.
func uniqueEmail(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d@Example.COM", prefix, time.Now().UnixNano())
}

// cleanupEmail deletes any user whose email matches (case-insensitively).
func cleanupEmail(t *testing.T, email string) {
	t.Helper()
	pool := requireDB(t)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM users WHERE lower(email) = lower($1)", strings.TrimSpace(email))
	})
}

func countActiveRefreshTokens(t *testing.T, userID string) int {
	t.Helper()
	var n int
	err := requireDB(t).QueryRow(context.Background(),
		"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND revoked_at IS NULL",
		userID,
	).Scan(&n)
	if err != nil {
		t.Fatalf("counting refresh tokens: %v", err)
	}
	return n
}

// ─── 1. Email normalization ──────────────────────────────────────────────────

func TestAuthRegisterNormalizesEmail(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	raw := uniqueEmail(t, "Foo")
	cleanupEmail(t, raw)
	padded := "  " + raw + " "

	res, err := svc.Register(ctx, "Foo", padded, "password123", "America/Sao_Paulo", mustInvite(t, padded))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	want := strings.ToLower(raw)
	if res.User.Email != want {
		t.Fatalf("stored email = %q, want %q", res.User.Email, want)
	}

	if _, err := svc.Login(ctx, want, "password123"); err != nil {
		t.Fatalf("login with lowercase email: %v", err)
	}
	if _, err := svc.Login(ctx, " "+strings.ToUpper(raw)+" ", "password123"); err != nil {
		t.Fatalf("login with uppercase padded email: %v", err)
	}
}

func TestAuthRegisterRejectsCaseVariantDuplicate(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	raw := uniqueEmail(t, "Dup")
	cleanupEmail(t, raw)

	if _, err := svc.Register(ctx, "Dup", raw, "password123", "UTC", mustInvite(t, raw)); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := svc.Register(ctx, "Dup", strings.ToUpper(raw), "password123", "UTC", mustInvite(t, raw))
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("second register err = %v, want ErrEmailAlreadyExists", err)
	}
}

// ─── 2. Register race → 409 ──────────────────────────────────────────────────

func TestIsUniqueViolation(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain", errors.New("boom"), false},
		{"unique", &pgconn.PgError{Code: "23505"}, true},
		{"wrapped unique", fmt.Errorf("insert: %w", &pgconn.PgError{Code: "23505"}), true},
		{"fk violation", &pgconn.PgError{Code: "23503"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isUniqueViolation(tc.err); got != tc.want {
				t.Fatalf("isUniqueViolation(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestAuthRegisterConcurrentSameEmail(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	email := uniqueEmail(t, "race")
	cleanupEmail(t, email)
	invite := mustInvite(t, email) // every attempt races on the same link

	const n = 6
	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = svc.Register(ctx, "Race", email, "password123", "UTC", invite)
		}(i)
	}
	close(start)
	wg.Wait()

	created := 0
	for i, err := range errs {
		switch {
		case err == nil:
			created++
		case errors.Is(err, ErrEmailAlreadyExists):
		default:
			t.Errorf("register #%d: unexpected error %v (would be a 500)", i, err)
		}
	}
	if created != 1 {
		t.Fatalf("created = %d, want exactly 1", created)
	}
}

// ─── 3. ChangePassword revokes refresh tokens ────────────────────────────────

func TestChangePasswordRevokesRefreshTokens(t *testing.T) {
	svc := newTestAuthService(t)
	users := NewUserService(requireDB(t), testLogger)
	ctx := context.Background()

	email := uniqueEmail(t, "pw")
	cleanupEmail(t, email)

	reg, err := svc.Register(ctx, "Pw", email, "password123", "UTC", mustInvite(t, email))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	login, err := svc.Login(ctx, email, "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if err := users.ChangePassword(ctx, reg.User.ID, "password123", "newpassword456"); err != nil {
		t.Fatalf("change password: %v", err)
	}

	for name, tok := range map[string]string{"register": reg.RefreshToken, "login": login.RefreshToken} {
		if _, err := svc.Refresh(ctx, tok); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("refresh with pre-change %s token err = %v, want ErrInvalidCredentials", name, err)
		}
	}
	if n := countActiveRefreshTokens(t, reg.User.ID); n != 0 {
		t.Fatalf("active refresh tokens after password change = %d, want 0", n)
	}

	// The new password still works and yields a usable refresh token.
	fresh, err := svc.Login(ctx, email, "newpassword456")
	if err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	if _, err := svc.Refresh(ctx, fresh.RefreshToken); err != nil {
		t.Fatalf("refresh with post-change token: %v", err)
	}
}

// ─── 4. Refresh token reuse detection ────────────────────────────────────────

func TestRefreshRotatesValidToken(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	email := uniqueEmail(t, "rot")
	cleanupEmail(t, email)

	reg, err := svc.Register(ctx, "Rot", email, "password123", "UTC", mustInvite(t, email))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	next, err := svc.Refresh(ctx, reg.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if next.RefreshToken == reg.RefreshToken {
		t.Fatal("refresh did not rotate the token")
	}
	if n := countActiveRefreshTokens(t, reg.User.ID); n != 1 {
		t.Fatalf("active tokens after rotation = %d, want 1", n)
	}
}

func TestRefreshReuseRevokesAllUserTokens(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	email := uniqueEmail(t, "reuse")
	cleanupEmail(t, email)

	reg, err := svc.Register(ctx, "Reuse", email, "password123", "UTC", mustInvite(t, email)) // token A
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	other, err := svc.Login(ctx, email, "password123") // token B (another device)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	rotated, err := svc.Refresh(ctx, reg.RefreshToken) // A revoked, C issued
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	// Replaying A (already revoked) must be rejected and burn every session.
	if _, err := svc.Refresh(ctx, reg.RefreshToken); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("replayed refresh err = %v, want ErrInvalidCredentials", err)
	}
	if n := countActiveRefreshTokens(t, reg.User.ID); n != 0 {
		t.Fatalf("active tokens after reuse = %d, want 0", n)
	}
	for name, tok := range map[string]string{"other device": other.RefreshToken, "rotated": rotated.RefreshToken} {
		if _, err := svc.Refresh(ctx, tok); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("refresh with %s token after reuse err = %v, want ErrInvalidCredentials", name, err)
		}
	}
}

func TestRefreshUnknownTokenDoesNotRevokeOthers(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	email := uniqueEmail(t, "unknown")
	cleanupEmail(t, email)

	reg, err := svc.Register(ctx, "Unknown", email, "password123", "UTC", mustInvite(t, email))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.Refresh(ctx, "not-a-real-token"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown token err = %v, want ErrInvalidCredentials", err)
	}
	if n := countActiveRefreshTokens(t, reg.User.ID); n != 1 {
		t.Fatalf("active tokens after unknown refresh = %d, want 1", n)
	}
}

// ─── 5. Config validation ────────────────────────────────────────────────────

func TestConfigLoadValidatesJWTExpiry(t *testing.T) {
	cases := []struct {
		name, hours, days string
		wantErr           bool
	}{
		{"defaults ok", "", "", false},
		{"explicit ok", "12", "7", false},
		{"hours not numeric", "abc", "30", true},
		{"hours zero", "0", "30", true},
		{"hours negative", "-1", "30", true},
		{"days not numeric", "24", "abc", true},
		{"days zero", "24", "0", true},
		{"hours float", "1.5", "30", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// production skips .env loading so the test is hermetic.
			t.Setenv("APP_ENV", "production")
			t.Setenv("JWT_SECRET", testJWTSecret)
			t.Setenv("JWT_EXPIRY_HOURS", tc.hours)
			t.Setenv("JWT_REFRESH_EXPIRY_DAYS", tc.days)

			_, err := config.Load()
			if tc.wantErr && err == nil {
				t.Fatal("config.Load() succeeded, want error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("config.Load() err = %v, want nil", err)
			}
		})
	}
}
