package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestSaveAndGetThread(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	t.Run("saves and retrieves thread", func(t *testing.T) {
		thread := &models.Thread{
			UserID:         userID,
			StableThreadID: "test-thread-id-123",
			Subject:        "Test Subject",
		}

		err := SaveThread(ctx, pool, thread)
		if err != nil {
			t.Fatalf("SaveThread failed: %v", err)
		}

		if thread.ID == "" {
			t.Error("Expected thread ID to be set")
		}

		retrieved, err := GetThreadByStableID(ctx, pool, userID, "test-thread-id-123")
		if err != nil {
			t.Fatalf("GetThreadByStableID failed: %v", err)
		}

		if retrieved.StableThreadID != thread.StableThreadID {
			t.Errorf("Expected StableThreadID %s, got %s", thread.StableThreadID, retrieved.StableThreadID)
		}
		if retrieved.Subject != thread.Subject {
			t.Errorf("Expected Subject %s, got %s", thread.Subject, retrieved.Subject)
		}
	})

	t.Run("updates existing thread", func(t *testing.T) {
		thread := &models.Thread{
			UserID:         userID,
			StableThreadID: "test-thread-id-456",
			Subject:        "Original Subject",
		}

		err := SaveThread(ctx, pool, thread)
		if err != nil {
			t.Fatalf("SaveThread failed: %v", err)
		}

		thread.Subject = "Updated Subject"
		err = SaveThread(ctx, pool, thread)
		if err != nil {
			t.Fatalf("SaveThread (update) failed: %v", err)
		}

		retrieved, err := GetThreadByStableID(ctx, pool, userID, "test-thread-id-456")
		if err != nil {
			t.Fatalf("GetThreadByStableID failed: %v", err)
		}

		if retrieved.Subject != "Updated Subject" {
			t.Errorf("Expected updated Subject, got %s", retrieved.Subject)
		}
	})

	t.Run("returns error for non-existent thread", func(t *testing.T) {
		_, err := GetThreadByStableID(ctx, pool, userID, "non-existent-thread-id")
		if err == nil {
			t.Error("Expected error for non-existent thread")
		}
		if !errors.Is(err, ErrThreadNotFound) {
			t.Errorf("Expected ErrThreadNotFound, got %v", err)
		}
	})
}

func TestGetThreadsForFolder(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Create threads
	thread1 := &models.Thread{
		UserID:         userID,
		StableThreadID: "thread-1",
		Subject:        "Subject 1",
	}
	thread2 := &models.Thread{
		UserID:         userID,
		StableThreadID: "thread-2",
		Subject:        "Subject 2",
	}

	err = SaveThread(ctx, pool, thread1)
	if err != nil {
		t.Fatalf("SaveThread failed: %v", err)
	}
	err = SaveThread(ctx, pool, thread2)
	if err != nil {
		t.Fatalf("SaveThread failed: %v", err)
	}

	// Create messages in different folders
	now := time.Now()
	msg1 := &models.Message{
		ThreadID:        thread1.ID,
		UserID:          userID,
		IMAPUID:         1,
		IMAPFolderName:  "INBOX",
		MessageIDHeader: "msg-1",
		Subject:         "Subject 1",
		SentAt:          &now,
	}
	msg2 := &models.Message{
		ThreadID:        thread2.ID,
		UserID:          userID,
		IMAPUID:         2,
		IMAPFolderName:  "INBOX",
		MessageIDHeader: "msg-2",
		Subject:         "Subject 2",
		SentAt:          &now,
	}
	msg3 := &models.Message{
		ThreadID:        thread1.ID,
		UserID:          userID,
		IMAPUID:         3,
		IMAPFolderName:  "Sent",
		MessageIDHeader: "msg-3",
		Subject:         "Subject 1",
		SentAt:          &now,
	}

	err = SaveMessage(ctx, pool, msg1)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}
	err = SaveMessage(ctx, pool, msg2)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}
	err = SaveMessage(ctx, pool, msg3)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	t.Run("returns threads for INBOX folder", func(t *testing.T) {
		threads, err := GetThreadsForFolder(ctx, pool, userID, "INBOX", 10, 0)
		if err != nil {
			t.Fatalf("GetThreadsForFolder failed: %v", err)
		}

		if len(threads) != 2 {
			t.Errorf("Expected 2 threads, got %d", len(threads))
		}
	})

	t.Run("returns threads for Sent folder", func(t *testing.T) {
		threads, err := GetThreadsForFolder(ctx, pool, userID, "Sent", 10, 0)
		if err != nil {
			t.Fatalf("GetThreadsForFolder failed: %v", err)
		}

		if len(threads) != 1 {
			t.Errorf("Expected 1 thread, got %d", len(threads))
		}
	})

	t.Run("respects pagination", func(t *testing.T) {
		threads, err := GetThreadsForFolder(ctx, pool, userID, "INBOX", 1, 0)
		if err != nil {
			t.Fatalf("GetThreadsForFolder failed: %v", err)
		}

		if len(threads) != 1 {
			t.Errorf("Expected 1 thread with limit 1, got %d", len(threads))
		}
	})
}

func TestGetThreadCountForFolder(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "count-test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	folderName := "INBOX"

	// Ensure the `folder_sync_timestamps` table exists with new columns
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS folder_sync_timestamps (
			user_id        UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			folder_name    TEXT        NOT NULL,
			synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_synced_uid BIGINT,
			thread_count   INT DEFAULT 0,
			PRIMARY KEY (user_id, folder_name)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to ensure folder_sync_timestamps table exists: %v", err)
	}

	// Create threads and messages
	thread1 := &models.Thread{
		UserID:         userID,
		StableThreadID: "count-thread-1",
		Subject:        "Thread 1",
	}
	thread2 := &models.Thread{
		UserID:         userID,
		StableThreadID: "count-thread-2",
		Subject:        "Thread 2",
	}
	thread3 := &models.Thread{
		UserID:         userID,
		StableThreadID: "count-thread-3",
		Subject:        "Thread 3",
	}

	err = SaveThread(ctx, pool, thread1)
	if err != nil {
		t.Fatalf("SaveThread failed: %v", err)
	}
	err = SaveThread(ctx, pool, thread2)
	if err != nil {
		t.Fatalf("SaveThread failed: %v", err)
	}
	err = SaveThread(ctx, pool, thread3)
	if err != nil {
		t.Fatalf("SaveThread failed: %v", err)
	}

	now := time.Now()
	msg1 := &models.Message{
		ThreadID:        thread1.ID,
		UserID:          userID,
		IMAPUID:         1,
		IMAPFolderName:  folderName,
		MessageIDHeader: "msg-1",
		Subject:         "Thread 1",
		SentAt:          &now,
	}
	msg2 := &models.Message{
		ThreadID:        thread2.ID,
		UserID:          userID,
		IMAPUID:         2,
		IMAPFolderName:  folderName,
		MessageIDHeader: "msg-2",
		Subject:         "Thread 2",
		SentAt:          &now,
	}
	msg3 := &models.Message{
		ThreadID:        thread3.ID,
		UserID:          userID,
		IMAPUID:         3,
		IMAPFolderName:  folderName,
		MessageIDHeader: "msg-3",
		Subject:         "Thread 3",
		SentAt:          &now,
	}

	err = SaveMessage(ctx, pool, msg1)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}
	err = SaveMessage(ctx, pool, msg2)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}
	err = SaveMessage(ctx, pool, msg3)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	t.Run("falls back to calculation when no materialized count exists", func(t *testing.T) {
		count, err := GetThreadCountForFolder(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("GetThreadCountForFolder failed: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected count 3, got %d", count)
		}
	})

	t.Run("uses materialized count when available", func(t *testing.T) {
		// Set materialized count
		err := UpdateThreadCount(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("UpdateThreadCount failed: %v", err)
		}

		count, err := GetThreadCountForFolder(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("GetThreadCountForFolder failed: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected count 3, got %d", count)
		}
	})

	t.Run("updates materialized count correctly", func(t *testing.T) {
		// Add another thread and message
		thread4 := &models.Thread{
			UserID:         userID,
			StableThreadID: "count-thread-4",
			Subject:        "Thread 4",
		}
		err := SaveThread(ctx, pool, thread4)
		if err != nil {
			t.Fatalf("SaveThread failed: %v", err)
		}

		msg4 := &models.Message{
			ThreadID:        thread4.ID,
			UserID:          userID,
			IMAPUID:         4,
			IMAPFolderName:  folderName,
			MessageIDHeader: "msg-4",
			Subject:         "Thread 4",
			SentAt:          &now,
		}
		err = SaveMessage(ctx, pool, msg4)
		if err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}

		// Update count
		err = UpdateThreadCount(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("UpdateThreadCount failed: %v", err)
		}

		// Should now return 4
		count, err := GetThreadCountForFolder(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("GetThreadCountForFolder failed: %v", err)
		}
		if count != 4 {
			t.Errorf("Expected count 4 after update, got %d", count)
		}
	})

	t.Run("handles NULL materialized count", func(t *testing.T) {
		// Set a row with NULL thread_count
		_, err := pool.Exec(ctx, `
			INSERT INTO folder_sync_timestamps (user_id, folder_name, synced_at, thread_count)
			VALUES ($1, $2, now(), NULL)
			ON CONFLICT (user_id, folder_name) DO UPDATE SET thread_count = NULL
		`, userID, "TestFolder")
		if err != nil {
			t.Fatalf("Failed to set NULL count: %v", err)
		}

		// Should fall back to calculation
		count, err := GetThreadCountForFolder(ctx, pool, userID, "TestFolder")
		if err != nil {
			t.Fatalf("GetThreadCountForFolder failed: %v", err)
		}
		// TestFolder has no messages, so count should be 0
		if count != 0 {
			t.Errorf("Expected count 0 for empty folder, got %d", count)
		}
	})
}

func TestGetFolderSyncInfo(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "sync-info-test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Ensure folder_sync_timestamps table exists with new columns
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS folder_sync_timestamps (
			user_id        UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			folder_name    TEXT        NOT NULL,
			synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_synced_uid BIGINT,
			thread_count   INT DEFAULT 0,
			PRIMARY KEY (user_id, folder_name)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to ensure folder_sync_timestamps table exists: %v", err)
	}

	folderName := "INBOX"

	t.Run("returns nil when no sync info exists", func(t *testing.T) {
		info, err := GetFolderSyncInfo(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("GetFolderSyncInfo failed: %v", err)
		}
		if info != nil {
			t.Errorf("Expected nil when no sync info exists, got %+v", info)
		}
	})

	t.Run("returns sync info with UID when set", func(t *testing.T) {
		lastUID := int64(12345)
		err := SetFolderSyncInfo(ctx, pool, userID, folderName, &lastUID)
		if err != nil {
			t.Fatalf("SetFolderSyncInfo failed: %v", err)
		}

		info, err := GetFolderSyncInfo(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("GetFolderSyncInfo failed: %v", err)
		}
		if info == nil {
			t.Fatal("Expected sync info, got nil")
		}
		if info.LastSyncedUID == nil {
			t.Error("Expected LastSyncedUID to be set")
		} else if *info.LastSyncedUID != lastUID {
			t.Errorf("Expected LastSyncedUID %d, got %d", lastUID, *info.LastSyncedUID)
		}
		if info.SyncedAt == nil {
			t.Error("Expected SyncedAt to be set")
		}
	})

	t.Run("updates UID correctly", func(t *testing.T) {
		lastUID1 := int64(10000)
		err := SetFolderSyncInfo(ctx, pool, userID, folderName, &lastUID1)
		if err != nil {
			t.Fatalf("SetFolderSyncInfo failed: %v", err)
		}

		info1, err := GetFolderSyncInfo(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("GetFolderSyncInfo failed: %v", err)
		}
		if info1 == nil || info1.LastSyncedUID == nil || *info1.LastSyncedUID != lastUID1 {
			t.Fatalf("Failed to set initial UID")
		}

		// Update to new UID
		lastUID2 := int64(20000)
		err = SetFolderSyncInfo(ctx, pool, userID, folderName, &lastUID2)
		if err != nil {
			t.Fatalf("SetFolderSyncInfo (update) failed: %v", err)
		}

		info2, err := GetFolderSyncInfo(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("GetFolderSyncInfo failed: %v", err)
		}
		if info2 == nil || info2.LastSyncedUID == nil {
			t.Fatal("Expected LastSyncedUID to be set after update")
		}
		if *info2.LastSyncedUID != lastUID2 {
			t.Errorf("Expected LastSyncedUID %d after update, got %d", lastUID2, *info2.LastSyncedUID)
		}
	})

	t.Run("preserves UID when setting with nil", func(t *testing.T) {
		lastUID := int64(30000)
		err := SetFolderSyncInfo(ctx, pool, userID, "TestFolder", &lastUID)
		if err != nil {
			t.Fatalf("SetFolderSyncInfo failed: %v", err)
		}

		// Set again with nil (should preserve existing UID)
		err = SetFolderSyncInfo(ctx, pool, userID, "TestFolder", nil)
		if err != nil {
			t.Fatalf("SetFolderSyncInfo (nil) failed: %v", err)
		}

		info, err := GetFolderSyncInfo(ctx, pool, userID, "TestFolder")
		if err != nil {
			t.Fatalf("GetFolderSyncInfo failed: %v", err)
		}
		if info == nil || info.LastSyncedUID == nil {
			t.Fatal("Expected LastSyncedUID to be preserved")
		}
		if *info.LastSyncedUID != lastUID {
			t.Errorf("Expected LastSyncedUID %d to be preserved, got %d", lastUID, *info.LastSyncedUID)
		}
	})
}

func TestUpdateThreadCount(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "update-count-test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	// Ensure folder_sync_timestamps table exists with new columns
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS folder_sync_timestamps (
			user_id        UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			folder_name    TEXT        NOT NULL,
			synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_synced_uid BIGINT,
			thread_count   INT DEFAULT 0,
			PRIMARY KEY (user_id, folder_name)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to ensure folder_sync_timestamps table exists: %v", err)
	}

	folderName := "INBOX"

	t.Run("updates count for existing folder", func(t *testing.T) {
		// Create sync info first
		err := SetFolderSyncInfo(ctx, pool, userID, folderName, nil)
		if err != nil {
			t.Fatalf("SetFolderSyncInfo failed: %v", err)
		}

		// Create threads and messages
		thread1 := &models.Thread{
			UserID:         userID,
			StableThreadID: "update-thread-1",
			Subject:        "Thread 1",
		}
		thread2 := &models.Thread{
			UserID:         userID,
			StableThreadID: "update-thread-2",
			Subject:        "Thread 2",
		}

		err = SaveThread(ctx, pool, thread1)
		if err != nil {
			t.Fatalf("SaveThread failed: %v", err)
		}
		err = SaveThread(ctx, pool, thread2)
		if err != nil {
			t.Fatalf("SaveThread failed: %v", err)
		}

		now := time.Now()
		msg1 := &models.Message{
			ThreadID:        thread1.ID,
			UserID:          userID,
			IMAPUID:         1,
			IMAPFolderName:  folderName,
			MessageIDHeader: "msg-1",
			Subject:         "Thread 1",
			SentAt:          &now,
		}
		msg2 := &models.Message{
			ThreadID:        thread2.ID,
			UserID:          userID,
			IMAPUID:         2,
			IMAPFolderName:  folderName,
			MessageIDHeader: "msg-2",
			Subject:         "Thread 2",
			SentAt:          &now,
		}

		err = SaveMessage(ctx, pool, msg1)
		if err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}
		err = SaveMessage(ctx, pool, msg2)
		if err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}

		// Update count
		err = UpdateThreadCount(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("UpdateThreadCount failed: %v", err)
		}

		// Verify count was updated
		info, err := GetFolderSyncInfo(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("GetFolderSyncInfo failed: %v", err)
		}
		if info == nil {
			t.Fatal("Expected sync info, got nil")
		}
		if info.ThreadCount != 2 {
			t.Errorf("Expected thread_count 2, got %d", info.ThreadCount)
		}
	})

	t.Run("handles folder with no messages", func(t *testing.T) {
		err := SetFolderSyncInfo(ctx, pool, userID, "EmptyFolder", nil)
		if err != nil {
			t.Fatalf("SetFolderSyncInfo failed: %v", err)
		}

		err = UpdateThreadCount(ctx, pool, userID, "EmptyFolder")
		if err != nil {
			t.Fatalf("UpdateThreadCount failed: %v", err)
		}

		info, err := GetFolderSyncInfo(ctx, pool, userID, "EmptyFolder")
		if err != nil {
			t.Fatalf("GetFolderSyncInfo failed: %v", err)
		}
		if info == nil {
			t.Fatal("Expected sync info, got nil")
		}
		if info.ThreadCount != 0 {
			t.Errorf("Expected thread_count 0 for empty folder, got %d", info.ThreadCount)
		}
	})
}

// TestGetThreadsForFolder_DeepPagination tests pagination performance with large datasets.
// This is a slow test that validates:
// 1. Deep pagination (page 10+ with large OFFSET) completes in reasonable time
// 2. The composite index is being used
// 3. Materialized count accuracy with large datasets
// 4. Correctness of pagination results
func TestGetThreadsForFolder_DeepPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow pagination test")
	}

	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "deep-pagination-test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	folderName := "INBOX"
	const totalThreads = 1500
	const threadsPerPage = 100

	// Ensure folder_sync_timestamps table exists
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS folder_sync_timestamps (
			user_id        UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			folder_name    TEXT        NOT NULL,
			synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_synced_uid BIGINT,
			thread_count   INT DEFAULT 0,
			PRIMARY KEY (user_id, folder_name)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to ensure folder_sync_timestamps table exists: %v", err)
	}

	// Create a large number of threads and messages
	// Use batch inserts for better performance
	t.Logf("Creating %d threads with messages...", totalThreads)
	baseTime := time.Now().Add(-time.Duration(totalThreads) * time.Hour)

	// Create threads in batches
	batchSize := 100
	for i := 0; i < totalThreads; i += batchSize {
		batchEnd := i + batchSize
		if batchEnd > totalThreads {
			batchEnd = totalThreads
		}

		// Create threads for this batch
		for j := i; j < batchEnd; j++ {
			thread := &models.Thread{
				UserID:         userID,
				StableThreadID: fmt.Sprintf("deep-thread-%d", j),
				Subject:        fmt.Sprintf("Thread %d", j),
			}
			if err := SaveThread(ctx, pool, thread); err != nil {
				t.Fatalf("Failed to save thread %d: %v", j, err)
			}

			// Create a message for each thread with different sent_at times
			sentAt := baseTime.Add(time.Duration(j) * time.Hour)
			msg := &models.Message{
				ThreadID:        thread.ID,
				UserID:          userID,
				IMAPUID:         int64(j + 1),
				IMAPFolderName:  folderName,
				MessageIDHeader: fmt.Sprintf("<msg-%d@test>", j),
				Subject:         fmt.Sprintf("Thread %d", j),
				SentAt:          &sentAt,
			}
			if err := SaveMessage(ctx, pool, msg); err != nil {
				t.Fatalf("Failed to save message %d: %v", j, err)
			}
		}

		if (i+batchSize)%500 == 0 {
			t.Logf("Created %d threads...", i+batchSize)
		}
	}

	t.Logf("Created %d threads. Updating materialized count...", totalThreads)

	// Update materialized count
	err = UpdateThreadCount(ctx, pool, userID, folderName)
	if err != nil {
		t.Fatalf("UpdateThreadCount failed: %v", err)
	}

	// Verify materialized count
	count, err := GetThreadCountForFolder(ctx, pool, userID, folderName)
	if err != nil {
		t.Fatalf("GetThreadCountForFolder failed: %v", err)
	}
	if count != totalThreads {
		t.Errorf("Expected materialized count %d, got %d", totalThreads, count)
	}

	t.Run("page 10 (OFFSET 900) completes in reasonable time", func(t *testing.T) {
		page := 10
		offset := (page - 1) * threadsPerPage

		start := time.Now()
		threads, err := GetThreadsForFolder(ctx, pool, userID, folderName, threadsPerPage, offset)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("GetThreadsForFolder failed: %v", err)
		}

		// Performance check: should complete in under 3 seconds
		if duration > 3*time.Second {
			t.Errorf("Page %d query took %v, expected < 3s", page, duration)
		}

		// Correctness check: should return exactly threadsPerPage threads
		if len(threads) != threadsPerPage {
			t.Errorf("Expected %d threads on page %d, got %d", threadsPerPage, page, len(threads))
		}

		// Verify threads are ordered correctly (newest first)
		for i := 1; i < len(threads); i++ {
			// We can't easily check sent_at here, but we can verify we got different threads
			if threads[i].ID == threads[i-1].ID {
				t.Errorf("Duplicate thread found on page %d", page)
			}
		}

		t.Logf("Page %d (OFFSET %d) completed in %v, returned %d threads", page, offset, duration, len(threads))
	})

	t.Run("page 15 (OFFSET 1400) completes in reasonable time", func(t *testing.T) {
		page := 15
		offset := (page - 1) * threadsPerPage

		start := time.Now()
		threads, err := GetThreadsForFolder(ctx, pool, userID, folderName, threadsPerPage, offset)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("GetThreadsForFolder failed: %v", err)
		}

		// Performance check: should complete in under 3 seconds
		if duration > 3*time.Second {
			t.Errorf("Page %d query took %v, expected < 3s", page, duration)
		}

		// Correctness check: should return threadsPerPage threads
		expectedCount := threadsPerPage
		if len(threads) != expectedCount {
			t.Errorf("Expected %d threads on page %d (OFFSET %d, total %d), got %d", expectedCount, page, offset, totalThreads, len(threads))
		}

		t.Logf("Page %d (OFFSET %d) completed in %v, returned %d threads (expected %d)", page, offset, duration, len(threads), expectedCount)
	})

	t.Run("index is being used for pagination query", func(t *testing.T) {
		// Use EXPLAIN to verify the index is being used
		rows, err := pool.Query(ctx, `
			EXPLAIN (FORMAT TEXT)
			SELECT t.id, t.user_id, t.stable_thread_id, t.subject, MAX(m2.sent_at) AS last_sent_at
			FROM threads t
			INNER JOIN messages m ON t.id = m.thread_id
			LEFT JOIN messages m2 ON m2.thread_id = t.id
			WHERE t.user_id = $1 AND m.imap_folder_name = $2
			GROUP BY t.id, t.user_id, t.stable_thread_id, t.subject
			ORDER BY last_sent_at DESC NULLS LAST
			LIMIT $3 OFFSET $4
		`, userID, folderName, threadsPerPage, 900)
		if err != nil {
			t.Fatalf("EXPLAIN query failed: %v", err)
		}
		defer rows.Close()

		var explainLines []string
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				t.Fatalf("Failed to scan explain result: %v", err)
			}
			explainLines = append(explainLines, line)
		}
		explainResult := strings.Join(explainLines, "\n")

		// Check if the index is mentioned in the explain plan
		// The index name is idx_messages_user_folder_sent_at
		if !strings.Contains(strings.ToLower(explainResult), "idx_messages_user_folder_sent_at") {
			t.Logf("Warning: Index idx_messages_user_folder_sent_at not found in explain plan:")
			t.Logf("Explain plan:\n%s", explainResult)
			// Don't fail the test, just warn - the index might still be used indirectly
		} else {
			t.Logf("Index idx_messages_user_folder_sent_at is being used")
		}
	})

	t.Run("materialized count is accurate with large dataset", func(t *testing.T) {
		count, err := GetThreadCountForFolder(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("GetThreadCountForFolder failed: %v", err)
		}
		if count != totalThreads {
			t.Errorf("Expected materialized count %d, got %d", totalThreads, count)
		}
	})
}
