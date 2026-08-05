package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/db"
)

// testPool is the shared connection to the throwaway Postgres container.
// Nil when Docker is unavailable — integration tests skip in that case so the
// unit tests still run (e.g. on a machine without Docker).
var testPool *pgxpool.Pool

var testLogger = zap.NewNop()

func TestMain(m *testing.M) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		os.Exit(m.Run())
	}

	ctx := context.Background()
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
		os.Exit(m.Run())
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "skipping integration tests: connection string: %v\n", err)
		os.Exit(m.Run())
	}

	// The same migration runner the API uses — tests exercise the real schema.
	if err := db.RunMigrations(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "migrations failed: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connecting to test db: %v\n", err)
		os.Exit(1)
	}
	testPool = pool

	code := m.Run()

	pool.Close()
	_ = testcontainers.TerminateContainer(container)
	os.Exit(code)
}

// requireDB skips a test when no container is available.
func requireDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testPool == nil {
		t.Skip("integration test requires Docker")
	}
	return testPool
}

// createTestUser inserts a user and returns its id. Each test gets its own so
// they never see each other's rows.
func createTestUser(t *testing.T) string {
	t.Helper()
	pool := requireDB(t)

	var userID string
	email := fmt.Sprintf("%s-%d@test.local", t.Name(), time.Now().UnixNano())
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email, password_hash, name) VALUES ($1, 'x', 'Test') RETURNING id::text`,
		email,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})
	return userID
}

func today() string {
	return time.Now().Format("2006-01-02")
}
