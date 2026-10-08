package ratelimit

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/myroutine/backend/internal/testutil"
)

var testPool *pgxpool.Pool

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

func requireDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testPool == nil {
		t.Skip("integration test requires Docker")
	}
	return testPool
}

func uniqueKey(t *testing.T) string {
	return fmt.Sprintf("203.0.113.%d-%s", time.Now().UnixNano()%250, t.Name())
}

func TestLimitAppliesPerKeyWithinTheMinute(t *testing.T) {
	l := NewPostgres(requireDB(t), 3)
	ctx := context.Background()
	key := uniqueKey(t)

	for i := 1; i <= 3; i++ {
		if ok, _, err := l.Allow(ctx, key); err != nil || !ok {
			t.Fatalf("attempt %d: ok=%v err=%v, want allowed", i, ok, err)
		}
	}
	ok, retry, err := l.Allow(ctx, key)
	if err != nil || ok {
		t.Fatalf("4th attempt: ok=%v err=%v, want blocked", ok, err)
	}
	if retry <= 0 || retry > time.Minute {
		t.Fatalf("retryAfter = %v, want within the current minute", retry)
	}
	if ok, _, _ := l.Allow(ctx, uniqueKey(t)+"-other"); !ok {
		t.Fatal("another client must have its own count")
	}
}

func TestWindowResetsInTheNextMinute(t *testing.T) {
	pool := requireDB(t)
	l := NewPostgres(pool, 1)
	ctx := context.Background()
	key := uniqueKey(t)

	l.Allow(ctx, key) //nolint:errcheck
	if ok, _, _ := l.Allow(ctx, key); ok {
		t.Fatal("2nd attempt with limit 1 must be blocked")
	}
	// Pretend the window started a minute ago.
	if _, err := pool.Exec(ctx,
		"UPDATE auth_rate_limits SET window_start = window_start - INTERVAL '1 minute' WHERE key_hash = $1",
		hashKey(key)); err != nil {
		t.Fatalf("aging window: %v", err)
	}
	if ok, _, err := l.Allow(ctx, key); err != nil || !ok {
		t.Fatalf("new minute: ok=%v err=%v, want allowed again", ok, err)
	}
}

func TestClientIPIsNeverStoredInClear(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	ip := "198.51.100.77"
	if _, _, err := NewPostgres(pool, 5).Allow(ctx, ip); err != nil {
		t.Fatalf("allow: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM auth_rate_limits WHERE key_hash LIKE '%' || $1 || '%'", ip).Scan(&n); err != nil || n != 0 {
		t.Fatalf("raw IP found in %d rows (err %v), want only hashes", n, err)
	}
}

func TestSweepDropsOnlyOldWindows(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	l := NewPostgres(pool, 5)
	oldKey, newKey := uniqueKey(t)+"-old", uniqueKey(t)+"-new"
	l.Allow(ctx, oldKey)                                                                                                         //nolint:errcheck
	l.Allow(ctx, newKey)                                                                                                         //nolint:errcheck
	pool.Exec(ctx, "UPDATE auth_rate_limits SET window_start = now() - INTERVAL '2 hours' WHERE key_hash = $1", hashKey(oldKey)) //nolint:errcheck

	l.sweep(ctx)

	var oldLeft, newLeft int
	pool.QueryRow(ctx, "SELECT count(*) FROM auth_rate_limits WHERE key_hash = $1", hashKey(oldKey)).Scan(&oldLeft) //nolint:errcheck
	pool.QueryRow(ctx, "SELECT count(*) FROM auth_rate_limits WHERE key_hash = $1", hashKey(newKey)).Scan(&newLeft) //nolint:errcheck
	if oldLeft != 0 || newLeft != 1 {
		t.Fatalf("after sweep: old rows %d (want 0), current rows %d (want 1)", oldLeft, newLeft)
	}
}

// Several API copies hitting at the same instant must not slip past the
// limit: the bump-and-read is one statement.
func TestConcurrentCopiesCannotExceedTheLimit(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	key := uniqueKey(t)
	copies := []*Postgres{NewPostgres(pool, 3), NewPostgres(pool, 3)}

	var allowed atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if ok, _, err := copies[i%2].Allow(ctx, key); err == nil && ok {
				allowed.Add(1)
			}
		}(i)
	}
	close(start)
	wg.Wait()

	if got := allowed.Load(); got != 3 {
		t.Fatalf("allowed %d of 20 concurrent attempts, want exactly 3", got)
	}
}
