package service

import (
	"context"
	"encoding/json"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/myroutine/backend/internal/appctx"
)

// Audit log: sensitive actions leave a row with who, what, IP and browser.

const auditTestIP = "203.0.113.50"

func auditCtx() context.Context {
	return appctx.WithRequestMeta(context.Background(), appctx.RequestMeta{IP: auditTestIP, UserAgent: "TestAgent/1"})
}

type auditRow struct {
	Action    string
	IP        *string
	UserAgent *string
	EntityID  *string
	Metadata  map[string]any
}

func auditRowsFor(t *testing.T, userID string) []auditRow {
	t.Helper()
	rows, err := requireDB(t).Query(context.Background(),
		`SELECT action, host(ip_address), user_agent, entity_id::text, metadata
		 FROM audit_logs WHERE user_id = $1 ORDER BY created_at, id`, userID)
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	defer rows.Close()

	var out []auditRow
	for rows.Next() {
		var r auditRow
		var meta []byte
		if err := rows.Scan(&r.Action, &r.IP, &r.UserAgent, &r.EntityID, &meta); err != nil {
			t.Fatalf("scan audit: %v", err)
		}
		if meta != nil {
			_ = json.Unmarshal(meta, &r.Metadata)
		}
		out = append(out, r)
	}
	return out
}

func findAudit(rows []auditRow, action string) *auditRow {
	for i := range rows {
		if rows[i].Action == action {
			return &rows[i]
		}
	}
	return nil
}

func registerAudited(t *testing.T, prefix string) (userID, email string) {
	t.Helper()
	email = uniqueEmail(t, prefix)
	cleanupEmail(t, email)
	res, err := newTestAuthService(t).Register(auditCtx(), "Aud", email, "senha-certa-1", "UTC", mustInvite(t, email))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return res.User.ID, email
}

func TestDummyPasswordHashIsRealBcrypt(t *testing.T) {
	// A malformed hash makes bcrypt fail instantly, so a login for an unknown
	// email would answer faster than for a real one — leaking which exist.
	cost, err := bcrypt.Cost(dummyPasswordHash)
	if err != nil || cost != 12 {
		t.Fatalf("dummy hash cost = %d, err = %v; want a valid cost-12 bcrypt hash", cost, err)
	}
}

func TestAuditLoginAndRegister(t *testing.T) {
	auth := newTestAuthService(t)
	userID, email := registerAudited(t, "audlogin")

	if _, err := auth.Login(auditCtx(), email, "senha-certa-1"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := auth.Login(auditCtx(), email, "senha-errada"); err == nil {
		t.Fatal("wrong password must fail")
	}

	rows := auditRowsFor(t, userID)
	reg := findAudit(rows, AuditRegister)
	if reg == nil || reg.Metadata["via"] != "invite" {
		t.Fatalf("register row = %+v, want via=invite", reg)
	}
	login := findAudit(rows, AuditLogin)
	if login == nil || login.IP == nil || *login.IP != auditTestIP || login.UserAgent == nil || *login.UserAgent != "TestAgent/1" {
		t.Fatalf("login row = %+v, want IP %s and user agent", login, auditTestIP)
	}
	if findAudit(rows, AuditLoginFailed) == nil {
		t.Fatal("wrong password must be recorded as auth.login_failed")
	}
}

func TestAuditFailedLoginForUnknownEmail(t *testing.T) {
	pool := requireDB(t)
	email := uniqueEmail(t, "ghost")
	if _, err := newTestAuthService(t).Login(auditCtx(), email, "qualquer-coisa"); err == nil {
		t.Fatal("unknown email must fail")
	}

	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs
		 WHERE action = $1 AND user_id IS NULL AND metadata->>'email' = $2`,
		AuditLoginFailed, normalizeEmail(email),
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM audit_logs WHERE metadata->>'email' = $1", normalizeEmail(email))
	})
	if n != 1 {
		t.Fatalf("failed-login rows for unknown email = %d, want 1 (no owner, email in metadata)", n)
	}
}

func TestAuditResetLinkIsOnTheAffectedPersonsHistory(t *testing.T) {
	pool := requireDB(t)
	adminID := createTestUser(t)
	targetID, email := registerAudited(t, "audreset")
	resets := NewPasswordResetService(pool)

	created, err := resets.Create(auditCtx(), adminID, email)
	if err != nil {
		t.Fatalf("create reset: %v", err)
	}
	if err := resets.Reset(auditCtx(), created.Token, "senha-nova-22"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	rows := auditRowsFor(t, targetID)
	link := findAudit(rows, AuditResetLinkCreated)
	if link == nil || link.Metadata["by_admin"] != adminID {
		t.Fatalf("reset-link row on target = %+v, want by_admin=%s", link, adminID)
	}
	if findAudit(rows, AuditPasswordReset) == nil {
		t.Fatal("the reset itself must be recorded as auth.password_reset")
	}

	// It comes with the person's own LGPD export.
	export, err := NewUserService(pool, testLogger).Export(context.Background(), targetID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	raw, _ := json.Marshal(export["audit_logs"])
	if !json.Valid(raw) || !containsAction(raw, AuditResetLinkCreated) {
		t.Fatalf("export audit_logs = %s, want the reset-link event", raw)
	}
}

func containsAction(raw []byte, action string) bool {
	var rows []struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return false
	}
	for _, r := range rows {
		if r.Action == action {
			return true
		}
	}
	return false
}

func TestAuditInviteCreatedAndRevokedByAdmin(t *testing.T) {
	adminID := createTestUser(t)
	invites := NewInviteService(requireDB(t))

	inv, err := invites.Create(auditCtx(), adminID, uniqueEmail(t, "audinv"))
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if err := invites.Revoke(auditCtx(), adminID, inv.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	rows := auditRowsFor(t, adminID)
	for _, action := range []string{AuditInviteCreated, AuditInviteRevoked} {
		r := findAudit(rows, action)
		if r == nil || r.EntityID == nil || *r.EntityID != inv.ID {
			t.Fatalf("%s row = %+v, want entity_id %s", action, r, inv.ID)
		}
	}
}

func TestAuditPasswordChangeAndExport(t *testing.T) {
	pool := requireDB(t)
	users := NewUserService(pool, testLogger)
	userID, _ := registerAudited(t, "audpw")

	if err := users.ChangePassword(auditCtx(), userID, "senha-certa-1", "senha-nova-22"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	export, err := users.Export(auditCtx(), userID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	rows := auditRowsFor(t, userID)
	if findAudit(rows, AuditPasswordChanged) == nil || findAudit(rows, AuditAccountExported) == nil {
		t.Fatalf("rows = %+v, want password_changed and exported", rows)
	}
	raw, _ := json.Marshal(export["audit_logs"])
	if containsAction(raw, AuditAccountExported) {
		t.Fatal("an export must not contain its own 'exported' row")
	}
}

func TestAuditSessionReuseDetected(t *testing.T) {
	auth := newTestAuthService(t)
	userID, email := registerAudited(t, "audreuse")

	login, err := auth.Login(auditCtx(), email, "senha-certa-1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := auth.Refresh(auditCtx(), login.RefreshToken); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if _, err := auth.Refresh(auditCtx(), login.RefreshToken); err == nil {
		t.Fatal("reusing a rotated refresh token must fail")
	}

	if findAudit(auditRowsFor(t, userID), AuditSessionReuseDetected) == nil {
		t.Fatal("token reuse must be recorded")
	}
}

func TestAuditAccountDeletionIsAnonymous(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()

	// A user agent unique to this test, to find this person's rows later.
	email := uniqueEmail(t, "auddel")
	cleanupEmail(t, email)
	ua := "Del/" + email
	reqCtx := appctx.WithRequestMeta(ctx, appctx.RequestMeta{IP: auditTestIP, UserAgent: ua})
	res, err := newTestAuthService(t).Register(reqCtx, "Del", email, "senha-certa-1", "UTC", mustInvite(t, email))
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	count := func() int {
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM audit_logs
			 WHERE action = $1 AND user_id IS NULL AND ip_address IS NULL AND user_agent IS NULL`,
			AuditAccountDeleted,
		).Scan(&n); err != nil {
			t.Fatalf("count: %v", err)
		}
		return n
	}
	before := count()

	if err := NewUserService(pool, testLogger).DeleteAccount(reqCtx, res.User.ID, "senha-certa-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if after := count(); after != before+1 {
		t.Fatalf("anonymous account.deleted rows: before %d, after %d; want +1", before, after)
	}

	var identifying int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE user_agent = $1", ua).Scan(&identifying); err != nil {
		t.Fatalf("count identifying: %v", err)
	}
	if identifying != 0 {
		t.Fatalf("%d audit rows still carry the deleted person's user agent", identifying)
	}
}
