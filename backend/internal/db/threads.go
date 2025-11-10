package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/models"
)

// ErrThreadNotFound is returned when a requested thread cannot be found.
var ErrThreadNotFound = errors.New("thread not found")

// SaveThread saves or updates a thread in the database.
func SaveThread(ctx context.Context, pool *pgxpool.Pool, thread *models.Thread) error {
	var threadID string

	err := pool.QueryRow(ctx, `
		INSERT INTO threads (user_id, stable_thread_id, subject)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, stable_thread_id) DO UPDATE SET
			subject = EXCLUDED.subject
		RETURNING id
	`, thread.UserID, thread.StableThreadID, thread.Subject).Scan(&threadID)

	if err != nil {
		return fmt.Errorf("failed to save thread: %w", err)
	}

	thread.ID = threadID
	return nil
}

// GetThreadByStableID returns a thread by its stable thread ID.
func GetThreadByStableID(ctx context.Context, pool *pgxpool.Pool, userID, stableThreadID string) (*models.Thread, error) {
	var thread models.Thread

	err := pool.QueryRow(ctx, `
		SELECT id, user_id, stable_thread_id, subject
		FROM threads
		WHERE user_id = $1 AND stable_thread_id = $2
	`, userID, stableThreadID).Scan(
		&thread.ID,
		&thread.UserID,
		&thread.StableThreadID,
		&thread.Subject,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrThreadNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	return &thread, nil
}

// GetThreadsForFolder returns threads for a specific folder.
// It returns threads that have at least one message in the specified folder.
func GetThreadsForFolder(ctx context.Context, pool *pgxpool.Pool, userID, folderName string, limit, offset int) ([]*models.Thread, error) {
	rows, err := pool.Query(ctx, `
        SELECT t.id, t.user_id, t.stable_thread_id, t.subject, MAX(m2.sent_at) AS last_sent_at
        FROM threads t
        INNER JOIN messages m ON t.id = m.thread_id
        LEFT JOIN messages m2 ON m2.thread_id = t.id
        WHERE t.user_id = $1 AND m.imap_folder_name = $2
        GROUP BY t.id, t.user_id, t.stable_thread_id, t.subject
        ORDER BY last_sent_at DESC NULLS LAST
        LIMIT $3 OFFSET $4
    `, userID, folderName, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to get threads: %w", err)
	}
	defer rows.Close()

	var threads []*models.Thread
	for rows.Next() {
		var thread models.Thread
		var _lastSentAt *time.Time
		if err := rows.Scan(
			&thread.ID,
			&thread.UserID,
			&thread.StableThreadID,
			&thread.Subject,
			&_lastSentAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan thread: %w", err)
		}
		threads = append(threads, &thread)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating threads: %w", err)
	}

	return threads, nil
}

// GetThreadCountForFolder returns the total count of threads for a specific folder.
func GetThreadCountForFolder(ctx context.Context, pool *pgxpool.Pool, userID, folderName string) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `
        SELECT COUNT(DISTINCT t.id)
        FROM threads t
        INNER JOIN messages m ON t.id = m.thread_id
        WHERE t.user_id = $1 AND m.imap_folder_name = $2
    `, userID, folderName).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("failed to get thread count: %w", err)
	}

	return count, nil
}

// GetFolderSyncTimestamp returns the timestamp when we last synced the given folder.
// Returns nil if we've never synced it.
func GetFolderSyncTimestamp(ctx context.Context, pool *pgxpool.Pool, userID, folderName string) (*time.Time, error) {
	var syncedAt *time.Time

	err := pool.QueryRow(ctx, `
		SELECT synced_at
		FROM folder_sync_timestamps
		WHERE user_id = $1 AND folder_name = $2
	`, userID, folderName).Scan(&syncedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get folder sync timestamp: %w", err)
	}

	return syncedAt, nil
}

// SetFolderSyncTimestamp sets the timestamp when we last synced the given folder.
func SetFolderSyncTimestamp(ctx context.Context, pool *pgxpool.Pool, userID, folderName string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO folder_sync_timestamps (user_id, folder_name, synced_at)
		VALUES ($1, $2, now())
		ON CONFLICT (user_id, folder_name) DO UPDATE SET
			synced_at = now()
	`, userID, folderName)

	if err != nil {
		return fmt.Errorf("failed to set folder sync timestamp: %w", err)
	}

	return nil
}
