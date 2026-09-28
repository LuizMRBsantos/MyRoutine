package service

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// Admin-issued password reset links: 1 hour, single use, newest wins, and a
// successful reset logs the account out everywhere.

func newResetFixture(t *testing.T) (*PasswordResetService, *AuthService, string, string) {
	t.Helper()
	pool := requireDB(t)
	auth := newTestAuthService(t)

	email := uniqueEmail(t, "reset")
	cleanupEmail(t, email)
	reg, err := auth.Register(context.Background(), "Esquecido", email, "senha-antiga-1", "UTC", mustInvite(t, email))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return NewPasswordResetService(pool), auth, email, reg.RefreshToken
}

func TestPasswordResetFlow(t *testing.T) {
	resets, auth, email, oldRefresh := newResetFixture(t)
	ctx := context.Background()
	adminID := createTestUser(t)

	created, err := resets.Create(ctx, adminID, email)
	if err != nil {
		t.Fatalf("create reset: %v", err)
	}
	if got, err := resets.Lookup(ctx, created.Token); err != nil || got != normalizeEmail(email) {
		t.Fatalf("lookup = %q, %v; want %q", got, err, normalizeEmail(email))
	}

	if err := resets.Reset(ctx, created.Token, "senha-nova-22"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	if _, err := auth.Login(ctx, email, "senha-nova-22"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	if _, err := auth.Login(ctx, email, "senha-antiga-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login with old password: err = %v, want ErrInvalidCredentials", err)
	}
	if _, err := auth.Refresh(ctx, oldRefresh); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old session after reset: err = %v, want ErrInvalidCredentials", err)
	}

	// The link works once.
	if err := resets.Reset(ctx, created.Token, "outra-senha-33"); !errors.Is(err, ErrResetInvalid) {
		t.Fatalf("reusing link: err = %v, want ErrResetInvalid", err)
	}
	if _, err := resets.Lookup(ctx, created.Token); !errors.Is(err, ErrResetInvalid) {
		t.Fatalf("lookup of used link: err = %v, want ErrResetInvalid", err)
	}
}

func TestNewResetLinkSupersedesOldOne(t *testing.T) {
	resets, _, email, _ := newResetFixture(t)
	ctx := context.Background()
	adminID := createTestUser(t)

	first, err := resets.Create(ctx, adminID, email)
	if err != nil {
		t.Fatalf("first reset: %v", err)
	}
	second, err := resets.Create(ctx, adminID, email)
	if err != nil {
		t.Fatalf("second reset: %v", err)
	}

	if err := resets.Reset(ctx, first.Token, "senha-nova-22"); !errors.Is(err, ErrResetInvalid) {
		t.Fatalf("superseded link: err = %v, want ErrResetInvalid", err)
	}
	if err := resets.Reset(ctx, second.Token, "senha-nova-22"); err != nil {
		t.Fatalf("newest link: %v", err)
	}
}

func TestExpiredResetLinkIsRefused(t *testing.T) {
	resets, _, email, _ := newResetFixture(t)
	pool := requireDB(t)
	ctx := context.Background()

	created, err := resets.Create(ctx, createTestUser(t), email)
	if err != nil {
		t.Fatalf("create reset: %v", err)
	}
	if _, err := pool.Exec(ctx,
		"UPDATE password_resets SET expires_at = NOW() - INTERVAL '1 minute' WHERE token_hash = $1",
		hashToken(created.Token),
	); err != nil {
		t.Fatalf("expiring reset: %v", err)
	}

	if err := resets.Reset(ctx, created.Token, "senha-nova-22"); !errors.Is(err, ErrResetInvalid) {
		t.Fatalf("err = %v, want ErrResetInvalid", err)
	}
}

func TestResetRejectsShortPasswordWithoutBurningLink(t *testing.T) {
	resets, _, email, _ := newResetFixture(t)
	ctx := context.Background()

	created, err := resets.Create(ctx, createTestUser(t), email)
	if err != nil {
		t.Fatalf("create reset: %v", err)
	}
	if err := resets.Reset(ctx, created.Token, "curta"); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("short password: err = %v, want ErrPasswordTooShort", err)
	}
	if err := resets.Reset(ctx, created.Token, "senha-nova-22"); err != nil {
		t.Fatalf("link must still work after a rejected short password: %v", err)
	}
}

func TestResetForUnknownEmail(t *testing.T) {
	resets := NewPasswordResetService(requireDB(t))
	if _, err := resets.Create(context.Background(), createTestUser(t), "ninguem@nada.test"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestResetLinkWorksOnceUnderConcurrency(t *testing.T) {
	resets, _, email, _ := newResetFixture(t)
	ctx := context.Background()

	created, err := resets.Create(ctx, createTestUser(t), email)
	if err != nil {
		t.Fatalf("create reset: %v", err)
	}

	const n = 4
	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = resets.Reset(ctx, created.Token, "senha-nova-22")
		}(i)
	}
	close(start)
	wg.Wait()

	ok := 0
	for i, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrResetInvalid):
		default:
			t.Errorf("reset #%d: unexpected error %v", i, err)
		}
	}
	if ok != 1 {
		t.Fatalf("successful resets = %d, want exactly 1", ok)
	}
}
