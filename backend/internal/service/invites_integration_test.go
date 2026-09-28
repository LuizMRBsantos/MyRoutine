package service

import (
	"context"
	"errors"
	"testing"

	"github.com/myroutine/backend/internal/config"
)

// Invite-only registration: an invite works once, only for its email, only
// while pending; admins listed in ADMIN_EMAILS register without one.

func newInviteServices(t *testing.T) (*InviteService, *AuthService, string) {
	t.Helper()
	pool := requireDB(t)
	return NewInviteService(pool), newTestAuthService(t), createTestUser(t)
}

func inviteStatus(t *testing.T, svc *InviteService, id string) string {
	t.Helper()
	invites, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("list invites: %v", err)
	}
	for _, inv := range invites {
		if inv.ID == id {
			return inv.Status
		}
	}
	t.Fatalf("invite %s not listed", id)
	return ""
}

func TestInviteRegistersInvitedEmailOnlyOnce(t *testing.T) {
	invites, auth, adminID := newInviteServices(t)
	ctx := context.Background()

	email := uniqueEmail(t, "invited")
	cleanupEmail(t, email)

	inv, err := invites.Create(ctx, adminID, email)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if got, err := invites.Lookup(ctx, inv.Token); err != nil || got != normalizeEmail(email) {
		t.Fatalf("lookup = %q, %v; want %q", got, err, normalizeEmail(email))
	}

	res, err := auth.Register(ctx, "Convidado", email, "password123", "UTC", inv.Token)
	if err != nil {
		t.Fatalf("register with invite: %v", err)
	}
	if res.User.IsAdmin {
		t.Fatal("an invited user must not be admin")
	}
	if got := inviteStatus(t, invites, inv.ID); got != "used" {
		t.Fatalf("status after use = %q, want used", got)
	}

	// The same link cannot create a second account, even for another email.
	other := uniqueEmail(t, "forwarded")
	cleanupEmail(t, other)
	if _, err := auth.Register(ctx, "Outro", other, "password123", "UTC", inv.Token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("reusing invite: err = %v, want ErrInviteInvalid", err)
	}
	if _, err := invites.Lookup(ctx, inv.Token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("lookup of used invite: err = %v, want ErrInviteInvalid", err)
	}
}

func TestRegisterWithoutInviteIsRefused(t *testing.T) {
	_, auth, _ := newInviteServices(t)

	email := uniqueEmail(t, "noinvite")
	cleanupEmail(t, email)
	if _, err := auth.Register(context.Background(), "X", email, "password123", "UTC", ""); !errors.Is(err, ErrInviteRequired) {
		t.Fatalf("err = %v, want ErrInviteRequired", err)
	}
}

func TestInviteOnlyWorksForItsEmail(t *testing.T) {
	invites, auth, adminID := newInviteServices(t)
	ctx := context.Background()

	invited := uniqueEmail(t, "right")
	intruder := uniqueEmail(t, "wrong")
	cleanupEmail(t, invited)
	cleanupEmail(t, intruder)

	inv, err := invites.Create(ctx, adminID, invited)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if _, err := auth.Register(ctx, "X", intruder, "password123", "UTC", inv.Token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("wrong email: err = %v, want ErrInviteInvalid", err)
	}
	// The failed attempt must not burn the invite for its real owner.
	if _, err := auth.Register(ctx, "Y", invited, "password123", "UTC", inv.Token); err != nil {
		t.Fatalf("real owner after failed attempt: %v", err)
	}
}

func TestExpiredInviteIsRefused(t *testing.T) {
	invites, auth, _ := newInviteServices(t)
	pool := requireDB(t)
	ctx := context.Background()

	email := uniqueEmail(t, "late")
	cleanupEmail(t, email)
	token := mustInvite(t, email)
	if _, err := pool.Exec(ctx,
		"UPDATE invites SET expires_at = NOW() - INTERVAL '1 minute' WHERE token_hash = $1",
		hashToken(token),
	); err != nil {
		t.Fatalf("expiring invite: %v", err)
	}

	if _, err := auth.Register(ctx, "X", email, "password123", "UTC", token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("err = %v, want ErrInviteInvalid", err)
	}
	var id string
	if err := pool.QueryRow(ctx, "SELECT id::text FROM invites WHERE token_hash = $1", hashToken(token)).Scan(&id); err != nil {
		t.Fatalf("reading invite id: %v", err)
	}
	if got := inviteStatus(t, invites, id); got != "expired" {
		t.Fatalf("status = %q, want expired", got)
	}
}

func TestRevokedInviteIsRefused(t *testing.T) {
	invites, auth, adminID := newInviteServices(t)
	ctx := context.Background()

	email := uniqueEmail(t, "revoked")
	cleanupEmail(t, email)
	inv, err := invites.Create(ctx, adminID, email)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if err := invites.Revoke(ctx, inv.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	if _, err := auth.Register(ctx, "X", email, "password123", "UTC", inv.Token); !errors.Is(err, ErrInviteInvalid) {
		t.Fatalf("err = %v, want ErrInviteInvalid", err)
	}
	if got := inviteStatus(t, invites, inv.ID); got != "revoked" {
		t.Fatalf("status = %q, want revoked", got)
	}
	if err := invites.Revoke(ctx, inv.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second revoke: err = %v, want ErrNotFound", err)
	}
	if err := invites.Revoke(ctx, "not-a-uuid"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("malformed id: err = %v, want ErrNotFound", err)
	}
}

func TestCreateInviteValidatesEmail(t *testing.T) {
	invites, auth, adminID := newInviteServices(t)
	ctx := context.Background()

	if _, err := invites.Create(ctx, adminID, "sem-arroba"); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("invalid email: err = %v, want ErrInvalidEmail", err)
	}

	// Someone who already has an account cannot be invited again.
	email := uniqueEmail(t, "member")
	cleanupEmail(t, email)
	if _, err := auth.Register(ctx, "Membro", email, "password123", "UTC", mustInvite(t, email)); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := invites.Create(ctx, adminID, email); !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("existing account: err = %v, want ErrEmailAlreadyExists", err)
	}
}

func TestAdminEmailRegistersWithoutInvite(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()

	email := uniqueEmail(t, "admin")
	cleanupEmail(t, email)
	auth := NewAuthService(&config.Config{
		JWTSecret:            testJWTSecret,
		JWTExpiryHours:       "24",
		JWTRefreshExpiryDays: "30",
		AdminEmails:          []string{normalizeEmail(email)},
	}, pool, testLogger)

	res, err := auth.Register(ctx, "Admin", email, "password123", "UTC", "")
	if err != nil {
		t.Fatalf("admin register without invite: %v", err)
	}
	if !res.User.IsAdmin {
		t.Fatal("admin email must register as admin")
	}

	login, err := auth.Login(ctx, email, "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !login.User.IsAdmin {
		t.Fatal("login must report is_admin")
	}
}

func TestPromoteAdminsUpgradesExistingAccount(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()

	email := uniqueEmail(t, "promote")
	cleanupEmail(t, email)
	if _, err := pool.Exec(ctx,
		"INSERT INTO users (email, password_hash, name) VALUES ($1, 'x', 'Old')", email,
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if err := PromoteAdmins(ctx, pool, []string{normalizeEmail(email)}); err != nil {
		t.Fatalf("promote: %v", err)
	}
	var isAdmin bool
	if err := pool.QueryRow(ctx, "SELECT is_admin FROM users WHERE email = $1", email).Scan(&isAdmin); err != nil {
		t.Fatalf("read is_admin: %v", err)
	}
	if !isAdmin {
		t.Fatal("existing account in ADMIN_EMAILS must be promoted")
	}
}
