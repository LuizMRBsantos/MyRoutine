package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
)

// LGPD: export must cover every table holding user data; deletion must erase
// everything and only with the right password.

func TestExportCoversEveryUserTable(t *testing.T) {
	pool := requireDB(t)

	rows, err := pool.Query(context.Background(),
		`SELECT table_name FROM information_schema.columns
		 WHERE table_schema = 'public' AND column_name = 'user_id'`)
	if err != nil {
		t.Fatalf("listing tables: %v", err)
	}
	defer rows.Close()

	known := map[string]bool{}
	for _, tbl := range exportTables {
		known[tbl] = true
	}
	for _, tbl := range exportExcludedTables {
		known[tbl] = true
	}

	var missing []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if !known[table] {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("tables with user_id not in exportTables nor exportExcludedTables: %v — "+
			"add them to the LGPD export (or justify excluding them)", missing)
	}
}

// accountFixture registers a user with a known password and some data.
func accountFixture(t *testing.T) (userID, email string) {
	t.Helper()
	pool := requireDB(t)
	ctx := context.Background()

	email = uniqueEmail(t, "lgpd")
	cleanupEmail(t, email)
	reg, err := newTestAuthService(t).Register(ctx, "Titular", email, "senha-certa-1", "UTC", mustInvite(t, email))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	userID = reg.User.ID

	if _, err := pool.Exec(ctx, "INSERT INTO habits (user_id, name) VALUES ($1, 'Correr')", userID); err != nil {
		t.Fatalf("insert habit: %v", err)
	}
	if _, err := NewFinanceService(pool, testLogger).CreateTransaction(ctx, userID, CreateTransactionInput{
		AmountCents: 1234, Category: "alimentacao", Description: "Almoço", OccurredOn: today(),
	}); err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO audit_logs (user_id, action, ip_address, user_agent) VALUES ($1, 'auth.login', '203.0.113.9', $2)",
		userID, "Safari/"+email,
	); err != nil {
		t.Fatalf("insert audit log: %v", err)
	}
	return userID, email
}

func TestExportContainsOnlyTheUsersData(t *testing.T) {
	pool := requireDB(t)
	users := NewUserService(pool, testLogger)
	userID, _ := accountFixture(t)
	otherID, _ := accountFixture(t)

	export, err := users.Export(context.Background(), userID)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	raw, err := json.Marshal(export)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(raw)

	if strings.Contains(body, "password_hash") || strings.Contains(body, "$2a$") {
		t.Fatal("export must never contain the password hash")
	}
	if strings.Contains(body, otherID) {
		t.Fatal("export leaked another user's data")
	}

	var decoded struct {
		Profile struct {
			ID string `json:"id"`
		} `json:"profile"`
		Habits []struct {
			Name string `json:"name"`
		} `json:"habits"`
		Transactions []struct {
			AmountCents int64 `json:"amount_cents"`
		} `json:"transactions"`
		BodyMetrics []any `json:"body_metrics"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Profile.ID != userID {
		t.Fatalf("profile id = %q, want %q", decoded.Profile.ID, userID)
	}
	if len(decoded.Habits) != 1 || decoded.Habits[0].Name != "Correr" {
		t.Fatalf("habits = %+v, want one 'Correr'", decoded.Habits)
	}
	if len(decoded.Transactions) != 1 || decoded.Transactions[0].AmountCents != 1234 {
		t.Fatalf("transactions = %+v, want one of 1234", decoded.Transactions)
	}
	if decoded.BodyMetrics == nil {
		t.Fatal("empty tables must export as [], not null")
	}
}

func TestDeleteAccountNeedsThePassword(t *testing.T) {
	pool := requireDB(t)
	users := NewUserService(pool, testLogger)
	userID, _ := accountFixture(t)

	if err := users.DeleteAccount(context.Background(), userID, "senha-errada"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: err = %v, want ErrInvalidCredentials", err)
	}
	var exists bool
	if err := pool.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID,
	).Scan(&exists); err != nil || !exists {
		t.Fatalf("account must survive a wrong password (exists=%v, err=%v)", exists, err)
	}
}

func TestDeleteAccountErasesEverything(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	users := NewUserService(pool, testLogger)
	userID, email := accountFixture(t)

	if err := users.DeleteAccount(ctx, userID, "senha-certa-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	for _, table := range append([]string{"users"}, exportTables...) {
		if table == "audit_logs" {
			continue // kept, but anonymized — checked below
		}
		col := "user_id"
		if table == "users" {
			col = "id"
		}
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE "+col+" = $1", userID).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 0 {
			t.Fatalf("%s still has %d rows of the deleted user", table, n)
		}
	}

	// The row survives (security history) but no longer identifies anyone.
	var kept, identifying int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE action = 'auth.login' AND user_id IS NULL
		   AND metadata IS NULL AND created_at > NOW() - INTERVAL '5 minutes'`,
	).Scan(&kept); err != nil || kept == 0 {
		t.Fatalf("audit row should be kept without owner (kept=%d, err=%v)", kept, err)
	}
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM audit_logs WHERE user_agent = $1", "Safari/"+email,
	).Scan(&identifying); err != nil {
		t.Fatalf("count audit logs: %v", err)
	}
	if identifying != 0 {
		t.Fatal("audit logs of the deleted user still carry IP / user agent")
	}
}

func TestProfileTimezoneIsValidated(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	users := NewUserService(pool, testLogger)
	userID := createTestUser(t)

	bad := "Marte/Olympus"
	if _, err := users.UpdateProfile(ctx, userID, UpdateProfileInput{Timezone: &bad}); !errors.Is(err, ErrInvalidTimezone) {
		t.Fatalf("invalid timezone: err = %v, want ErrInvalidTimezone", err)
	}
	good := "Europe/Lisbon"
	p, err := users.UpdateProfile(ctx, userID, UpdateProfileInput{Timezone: &good})
	if err != nil || p.Timezone != good {
		t.Fatalf("valid timezone: profile = %+v, err = %v", p, err)
	}
}

func TestRegisterFallsBackOnInvalidTimezone(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()

	email := uniqueEmail(t, "tz")
	cleanupEmail(t, email)
	res, err := newTestAuthService(t).Register(ctx, "Tz", email, "password123", "Marte/Olympus", mustInvite(t, email))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	var tz string
	if err := pool.QueryRow(ctx, "SELECT timezone FROM users WHERE id = $1", res.User.ID).Scan(&tz); err != nil {
		t.Fatalf("read timezone: %v", err)
	}
	if tz != "America/Sao_Paulo" {
		t.Fatalf("timezone = %q, want fallback America/Sao_Paulo", tz)
	}
}
