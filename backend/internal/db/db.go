package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a PostgreSQL connection pool using pgx.
// pgx is the most performant native Go PostgreSQL driver.
// Connect opens the pool. maxConns caps open connections: a long-running
// server can hold many, a serverless function (Vercel) must hold few — each
// instance has its own pool and Supabase's pooler has a connection budget.
func Connect(databaseURL string, maxConns int) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Pool tuning — important for production performance
	if maxConns < 1 {
		maxConns = 1
	}
	config.MaxConns = int32(maxConns)
	// Keep a few warm connections only when the pool is big (long-running
	// server); a small serverless pool opens connections on demand.
	config.MinConns = 0
	if maxConns >= 10 {
		config.MinConns = 5
	}
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection is actually alive
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
