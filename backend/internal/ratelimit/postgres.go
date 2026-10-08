// Package ratelimit counts attempts per client in Postgres, so every copy of
// the API (several run at once on Vercel) shares the same limit.
package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// sweepEvery: one in this many calls also deletes rows older than an hour.
const sweepEvery = 200

// Postgres is a fixed one-minute window per key, kept in auth_rate_limits
// (migration 016). The window follows the DATABASE clock, so all API copies
// agree on which minute it is.
type Postgres struct {
	db        *pgxpool.Pool
	perMinute int
	calls     atomic.Uint64
}

func NewPostgres(db *pgxpool.Pool, perMinute int) *Postgres {
	return &Postgres{db: db, perMinute: perMinute}
}

// Allow records one attempt for key and reports whether it is within the
// limit; when it is not, retryAfter is the time left in the current minute.
// The bump and the read happen in ONE statement, so concurrent copies of the
// API cannot both slip under the limit.
func (p *Postgres) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	var attempts int
	var secondsLeft float64
	err := p.db.QueryRow(ctx,
		`INSERT INTO auth_rate_limits (key_hash, window_start, attempts)
		 VALUES ($1, date_trunc('minute', now()), 1)
		 ON CONFLICT (key_hash) DO UPDATE SET
		   attempts = CASE WHEN auth_rate_limits.window_start = EXCLUDED.window_start
		                   THEN auth_rate_limits.attempts + 1 ELSE 1 END,
		   window_start = EXCLUDED.window_start
		 RETURNING attempts,
		   EXTRACT(EPOCH FROM (window_start + INTERVAL '1 minute' - now()))::float8`,
		hashKey(key),
	).Scan(&attempts, &secondsLeft)
	if err != nil {
		return false, 0, fmt.Errorf("rate limit: %w", err)
	}

	if p.calls.Add(1)%sweepEvery == 0 {
		p.sweep(ctx)
	}

	if attempts <= p.perMinute {
		return true, 0, nil
	}
	return false, time.Duration(math.Max(1, math.Ceil(secondsLeft))) * time.Second, nil
}

// sweep drops windows older than an hour. Best effort: a failure only means
// the table stays a bit bigger until the next sweep.
func (p *Postgres) sweep(ctx context.Context) {
	_, _ = p.db.Exec(ctx, `DELETE FROM auth_rate_limits WHERE window_start < now() - INTERVAL '1 hour'`)
}

// hashKey stores a fingerprint of the client IP, never the IP itself
// (personal data).
func hashKey(key string) string {
	sum := sha256.Sum256([]byte("auth-rate-limit:" + key))
	return hex.EncodeToString(sum[:])
}
