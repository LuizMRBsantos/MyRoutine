package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/myroutine/backend/internal/testutil"
)

// Migration 020 maps the old saturated colors to the new palette. The test
// DB already ran it, so re-run its SQL over rows written with old colors.
func TestSoftColorMigrationMapsOldPalette(t *testing.T) {
	ctx := context.Background()
	pool, cleanup, err := testutil.StartPostgres(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if pool == nil {
		t.Skip("integration test requires Docker")
	}

	var userID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ('cores@test.local', 'x', 'T') RETURNING id::text`,
	).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{"#0071E3", "#ff3b30", "#123456"} {
		if _, err := pool.Exec(ctx, `INSERT INTO habits (user_id, name, color) VALUES ($1, $2, $2)`, userID, c); err != nil {
			t.Fatal(err)
		}
	}

	up, err := os.ReadFile("migrations/000020_soft_habit_colors.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pool.Acquire(ctx) // pg_temp lives per connection
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, string(up)); err != nil {
		t.Fatalf("running migration 020 again: %v", err)
	}

	got := map[string]string{}
	rows, err := pool.Query(ctx, `SELECT name, color FROM habits WHERE user_id = $1`, userID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name, color string
		if err := rows.Scan(&name, &color); err != nil {
			t.Fatal(err)
		}
		got[name] = color
	}
	rows.Close()

	want := map[string]string{"#0071E3": "#7EA6D8", "#ff3b30": "#D894A8", "#123456": "#123456"}
	for name, color := range want {
		if got[name] != color {
			t.Errorf("habit %s: color %q, want %q", name, got[name], color)
		}
	}
}
