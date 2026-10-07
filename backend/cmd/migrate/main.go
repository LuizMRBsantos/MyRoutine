// Command migrate applies the embedded database migrations and exits. It is
// how the schema changes in production, where the API itself never migrates
// at startup (RUN_MIGRATIONS=false on Vercel): CI runs it before deploying.
//
//	MIGRATE_DATABASE_URL='postgres://...' go run ./cmd/migrate
//
// Use a connection that keeps one session for the whole run (Supabase:
// the "Session pooler", port 5432) — not the transaction pooler (6543).
package main

import (
	"fmt"
	"os"

	"github.com/myroutine/backend/internal/db"
)

func main() {
	url := os.Getenv("MIGRATE_DATABASE_URL")
	if url == "" {
		fmt.Fprintln(os.Stderr, "MIGRATE_DATABASE_URL is required")
		os.Exit(2)
	}
	if err := db.RunMigrations(url); err != nil {
		// The error may echo connection details, never print the URL itself.
		fmt.Fprintf(os.Stderr, "migrations failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("migrations applied")
}
