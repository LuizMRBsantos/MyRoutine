// Package testutil provides shared test infrastructure. It is imported only
// from _test.go files, never from production code.
package testutil

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/myroutine/backend/internal/db"
)

// StartPostgres boots a throwaway Postgres, applies the real migrations and
// returns a pool plus a cleanup func.
//
// The pool is nil when Docker is unavailable or SKIP_INTEGRATION is set —
// callers should skip their integration tests in that case so unit tests
// still run on machines without Docker.
func StartPostgres(ctx context.Context) (pool *pgxpool.Pool, cleanup func(), err error) {
	noop := func() {}

	if os.Getenv("SKIP_INTEGRATION") != "" {
		return nil, noop, nil
	}

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("myroutine_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skipping integration tests: could not start postgres: %v\n", err)
		return nil, noop, nil
	}

	terminate := func() { _ = testcontainers.TerminateContainer(container) }

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminate()
		return nil, noop, fmt.Errorf("connection string: %w", err)
	}

	// The same migration runner the API uses — tests exercise the real schema.
	if err := db.RunMigrations(dsn); err != nil {
		terminate()
		return nil, noop, fmt.Errorf("running migrations: %w", err)
	}

	// Production talks to Supabase's transaction pooler, which needs the simple
	// protocol (parameters sent as text). Tests use the same mode so a value
	// that only breaks there (e.g. []byte into jsonb) fails here first.
	pool, err = pgxpool.New(ctx, dsn+"&default_query_exec_mode=simple_protocol")
	if err != nil {
		terminate()
		return nil, noop, fmt.Errorf("connecting to test db: %w", err)
	}

	return pool, func() {
		pool.Close()
		terminate()
	}, nil
}
