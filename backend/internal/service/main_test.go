package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/testutil"
)

// testPool is the shared connection to the throwaway Postgres container.
// Nil when Docker is unavailable — integration tests skip in that case so the
// unit tests still run.
var testPool *pgxpool.Pool

var testLogger = zap.NewNop()

func TestMain(m *testing.M) {
	pool, cleanup, err := testutil.StartPostgres(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "test setup failed: %v\n", err)
		os.Exit(1)
	}
	testPool = pool

	code := m.Run()

	cleanup()
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
