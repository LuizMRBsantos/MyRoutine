package service

import (
	"context"
	"sort"
	"testing"
)

// Supabase's auto REST API reaches the public schema with its own roles.
// Every table must have RLS on (and no policies) so those roles see nothing.

func TestEveryTableHasRowLevelSecurity(t *testing.T) {
	rows, err := requireDB(t).Query(context.Background(),
		`SELECT c.relname FROM pg_class c
		 JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND NOT c.relrowsecurity`)
	if err != nil {
		t.Fatalf("listing tables: %v", err)
	}
	defer rows.Close()

	var open []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		open = append(open, name)
	}
	if len(open) > 0 {
		sort.Strings(open)
		t.Fatalf("tables without RLS: %v — add `ALTER TABLE ... ENABLE ROW LEVEL SECURITY` to their migration", open)
	}
}

// A role like Supabase's "anon", even when granted SELECT by mistake, must
// read nothing; the owner (our backend) keeps full access.
func TestRLSHidesRowsFromNonOwnerRoles(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	createTestUser(t) // at least one row exists

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck — nothing here may persist

	for _, stmt := range []string{
		"CREATE ROLE rls_probe NOLOGIN",
		"GRANT USAGE ON SCHEMA public TO rls_probe",
		"GRANT SELECT ON users TO rls_probe", // the "mistake"
	} {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}

	var asOwner int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&asOwner); err != nil || asOwner == 0 {
		t.Fatalf("owner sees %d users (err %v), want > 0", asOwner, err)
	}

	if _, err := tx.Exec(ctx, "SET LOCAL ROLE rls_probe"); err != nil {
		t.Fatalf("set role: %v", err)
	}
	var asProbe int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&asProbe); err != nil {
		t.Fatalf("probe query: %v", err)
	}
	if asProbe != 0 {
		t.Fatalf("a non-owner role read %d users through RLS, want 0", asProbe)
	}
}
