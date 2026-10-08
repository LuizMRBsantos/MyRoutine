package service

import (
	"context"
	"testing"
)

// Backups (pg_dump of schema public) must restore into a plain Postgres.
// Column defaults that call extension functions — uuid_generate_v4() lives
// in Supabase's "extensions" schema — break that restore; gen_random_uuid()
// is built into Postgres 13+.
func TestNoColumnDefaultDependsOnUUIDExtension(t *testing.T) {
	rows, err := requireDB(t).Query(context.Background(),
		`SELECT table_name || '.' || column_name
		 FROM information_schema.columns
		 WHERE table_schema = 'public' AND column_default LIKE '%uuid_generate_v4%'`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var bad []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			t.Fatalf("scan: %v", err)
		}
		bad = append(bad, col)
	}
	if len(bad) > 0 {
		t.Fatalf("columns defaulting to uuid_generate_v4(): %v — use gen_random_uuid()", bad)
	}
}
