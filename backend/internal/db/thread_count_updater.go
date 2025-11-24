package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ThreadCountUpdater defines an interface for updating thread counts.
// This allows the Service to be tested with mock implementations.
type ThreadCountUpdater interface {
	UpdateThreadCount(ctx context.Context, userID, folderName string) error
}

// threadCountUpdaterImpl implements ThreadCountUpdater using a database pool.
type threadCountUpdaterImpl struct {
	pool *pgxpool.Pool
}

// NewThreadCountUpdater creates a ThreadCountUpdater that uses the given database pool.
func NewThreadCountUpdater(pool *pgxpool.Pool) ThreadCountUpdater {
	return &threadCountUpdaterImpl{pool: pool}
}

// UpdateThreadCount updates the materialized thread count for a folder.
func (t *threadCountUpdaterImpl) UpdateThreadCount(ctx context.Context, userID, folderName string) error {
	return UpdateThreadCount(ctx, t.pool, userID, folderName)
}
