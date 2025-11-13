package db

import (
	"context"
	"errors"
	"fmt"
	"log"
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

// GetThreadByID returns a thread by its database ID.
func GetThreadByID(ctx context.Context, pool *pgxpool.Pool, threadID string) (*models.Thread, error) {
	var thread models.Thread

	err := pool.QueryRow(ctx, `
		SELECT id, user_id, stable_thread_id, subject
		FROM threads
		WHERE id = $1
	`, threadID).Scan(
		&thread.ID,
		&thread.UserID,
		&thread.StableThreadID,
		&thread.Subject,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrThreadNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get thread by ID: %w", err)
	}

	return &thread, nil
}

// GetThreadsForFolder returns threads for a specific folder.
// It returns threads that have at least one message in the specified folder.
func GetThreadsForFolder(ctx context.Context, pool *pgxpool.Pool, userID, folderName string, limit, offset int) ([]*models.Thread, error) {
	rows, err := pool.Query(ctx, `
        SELECT 
            t.id, 
            t.user_id, 
            t.stable_thread_id, 
            t.subject, 
            MAX(m2.sent_at) AS last_sent_at,
            (SELECT m3.from_address 
             FROM messages m3 
             WHERE m3.thread_id = t.id 
             ORDER BY m3.sent_at NULLS LAST 
             LIMIT 1) AS first_message_from_address
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
		var firstMessageFromAddress *string
		if err := rows.Scan(
			&thread.ID,
			&thread.UserID,
			&thread.StableThreadID,
			&thread.Subject,
			&_lastSentAt,
			&firstMessageFromAddress,
		); err != nil {
			return nil, fmt.Errorf("failed to scan thread: %w", err)
		}
		if firstMessageFromAddress != nil {
			thread.FirstMessageFromAddress = *firstMessageFromAddress
		}
		threads = append(threads, &thread)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating threads: %w", err)
	}

	return threads, nil
}

// GetThreadCountForFolder returns the total count of threads for a specific folder.
// Uses the materialized count from folder_sync_timestamps if available,
// otherwise falls back to calculating it on the fly.
func GetThreadCountForFolder(ctx context.Context, pool *pgxpool.Pool, userID, folderName string) (int, error) {
	// Try to get the materialized count first
	var count *int
	err := pool.QueryRow(ctx, `
		SELECT thread_count
		FROM folder_sync_timestamps
		WHERE user_id = $1 AND folder_name = $2
	`, userID, folderName).Scan(&count)

	if err == nil && count != nil {
		// Materialized count exists, use it
		return *count, nil
	}

	// If the row doesn't exist or the count is NULL, fallback to calculating on the fly
	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		// Some other error occurred, log it but continue to fallback
		log.Printf("Warning: Failed to get materialized thread count: %v", err)
	}

	// Fallback: calculate count on the fly
	var calculatedCount int
	err = pool.QueryRow(ctx, `
        SELECT COUNT(DISTINCT t.id)
        FROM threads t
        INNER JOIN messages m ON t.id = m.thread_id
        WHERE t.user_id = $1 AND m.imap_folder_name = $2
    `, userID, folderName).Scan(&calculatedCount)

	if err != nil {
		return 0, fmt.Errorf("failed to get thread count: %w", err)
	}

	return calculatedCount, nil
}

// FolderSyncInfo contains information about folder sync status.
type FolderSyncInfo struct {
	SyncedAt      *time.Time
	LastSyncedUID *int64
	ThreadCount   int
}

// GetFolderSyncInfo returns the sync information for the given folder.
// Returns nil if we've never synced it.
func GetFolderSyncInfo(ctx context.Context, pool *pgxpool.Pool, userID, folderName string) (*FolderSyncInfo, error) {
	var info FolderSyncInfo

	err := pool.QueryRow(ctx, `
		SELECT synced_at, last_synced_uid, thread_count
		FROM folder_sync_timestamps
		WHERE user_id = $1 AND folder_name = $2
	`, userID, folderName).Scan(&info.SyncedAt, &info.LastSyncedUID, &info.ThreadCount)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get folder sync info: %w", err)
	}

	return &info, nil
}

// SetFolderSyncInfo sets the sync information for the given folder.
func SetFolderSyncInfo(ctx context.Context, pool *pgxpool.Pool, userID, folderName string, lastSyncedUID *int64) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO folder_sync_timestamps (user_id, folder_name, synced_at, last_synced_uid)
		VALUES ($1, $2, now(), $3)
		ON CONFLICT (user_id, folder_name) DO UPDATE SET
			synced_at = now(),
			last_synced_uid = COALESCE($3, folder_sync_timestamps.last_synced_uid)
	`, userID, folderName, lastSyncedUID)

	if err != nil {
		return fmt.Errorf("failed to set folder sync info: %w", err)
	}

	return nil
}

// UpdateThreadCount updates the materialized thread count for a folder.
// This should be called in the background after syncing.
func UpdateThreadCount(ctx context.Context, pool *pgxpool.Pool, userID, folderName string) error {
	_, err := pool.Exec(ctx, `
		UPDATE folder_sync_timestamps
		SET thread_count = (
			SELECT COUNT(DISTINCT t.id)
			FROM threads t
			INNER JOIN messages m ON t.id = m.thread_id
			WHERE t.user_id = $1 AND m.imap_folder_name = $2
		)
		WHERE user_id = $1 AND folder_name = $2
	`, userID, folderName)

	if err != nil {
		return fmt.Errorf("failed to update thread count: %w", err)
	}

	return nil
}

// EnrichThreadsWithFirstMessageFromAddress enriches threads with the first message's from_address.
// This is useful for search results and other cases where threads don't have messages populated.
func EnrichThreadsWithFirstMessageFromAddress(ctx context.Context, pool *pgxpool.Pool, threads []*models.Thread) error {
	if len(threads) == 0 {
		return nil
	}

	// Build a map of thread IDs for efficient lookup
	threadIDMap := make(map[string]*models.Thread)
	threadIDs := make([]string, 0, len(threads))
	for _, thread := range threads {
		threadIDMap[thread.ID] = thread
		threadIDs = append(threadIDs, thread.ID)
	}

	// Query all first message from_addresses in one query
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT ON (thread_id) thread_id, from_address
		FROM messages
		WHERE thread_id = ANY($1)
		ORDER BY thread_id, sent_at NULLS LAST
	`, threadIDs)

	if err != nil {
		return fmt.Errorf("failed to get first message from addresses: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var threadID string
		var fromAddress string
		if err := rows.Scan(&threadID, &fromAddress); err != nil {
			return fmt.Errorf("failed to scan from address: %w", err)
		}
		if thread, exists := threadIDMap[threadID]; exists {
			thread.FirstMessageFromAddress = fromAddress
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating from addresses: %w", err)
	}

	return nil
}
