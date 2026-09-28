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

// mustInvite inserts a usable invite for email and returns its raw token, so
// tests can register through the real invite-only flow.
func mustInvite(t *testing.T, email string) string {
	t.Helper()
	pool := requireDB(t)

	token := fmt.Sprintf("test-invite-%d", time.Now().UnixNano())
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO invites (token_hash, email, expires_at)
		 VALUES ($1, $2, NOW() + INTERVAL '1 day') RETURNING id::text`,
		hashToken(token), normalizeEmail(email),
	).Scan(&id)
	if err != nil {
		t.Fatalf("creating test invite: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM invites WHERE id = $1", id)
	})
	return token
}

// today is the default user's local calendar day (America/Sao_Paulo, the
// zone a context without a timezone falls back to) — the same day the
// services default to when a test passes context.Background() and no date.
// Never the server clock: between 21:00 and 24:00 in São Paulo a UTC runner
// is already on the next day, and mixing the two would make tests flaky.
func today() string {
	return userToday(context.Background())
}
