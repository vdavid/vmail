package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/config"
)

// NewConnection creates a new PostgreSQL connection pool with the given configuration.
func NewConnection(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	dbURL := cfg.GetDatabaseURL()

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// CloseConnection closes the given database connection pool.
func CloseConnection(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
