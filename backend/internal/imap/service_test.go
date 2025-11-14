package imap

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

// TestShouldSyncFolder tests the cache TTL logic using a real database.
// This is an integration test that verifies the ShouldSyncFolder logic works correctly.
func TestShouldSyncFolder(t *testing.T) {
	// Setup: Use the test database (similar to other test files)
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	// Ensure the folder_sync_timestamps table exists (run migration if needed)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS folder_sync_timestamps (
			user_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			folder_name TEXT        NOT NULL,
			synced_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (user_id, folder_name)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to ensure folder_sync_timestamps table exists: %v", err)
	}

	encryptor := getTestEncryptor(t)
	service := NewService(pool, encryptor)
	defer service.Close()

	userID, err := db.GetOrCreateUser(ctx, pool, "sync-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	folderName := "INBOX"

	t.Run("returns true when no sync timestamp exists", func(t *testing.T) {
		shouldSync, err := service.ShouldSyncFolder(ctx, userID, folderName)
		if err != nil {
			t.Fatalf("ShouldSyncFolder failed: %v", err)
		}
		if !shouldSync {
			t.Error("Expected ShouldSyncFolder to return true when no timestamp exists")
		}
	})

	t.Run("returns false when cache is fresh", func(t *testing.T) {
		// Set a recent sync timestamp
		if err := db.SetFolderSyncInfo(ctx, pool, userID, folderName, nil); err != nil {
			t.Fatalf("Failed to set sync timestamp: %v", err)
		}

		shouldSync, err := service.ShouldSyncFolder(ctx, userID, folderName)
		if err != nil {
			t.Fatalf("ShouldSyncFolder failed: %v", err)
		}
		if shouldSync {
			t.Error("Expected ShouldSyncFolder to return false when cache is fresh")
		}
	})

	t.Run("returns true when cache is stale", func(t *testing.T) {
		// Manually set an old timestamp by updating the database directly
		_, err := pool.Exec(ctx, `
			UPDATE folder_sync_timestamps 
			SET synced_at = $1 
			WHERE user_id = $2 AND folder_name = $3
		`, time.Now().Add(-10*time.Minute), userID, folderName)
		if err != nil {
			t.Fatalf("Failed to set old timestamp: %v", err)
		}

		shouldSync, err := service.ShouldSyncFolder(ctx, userID, folderName)
		if err != nil {
			t.Fatalf("ShouldSyncFolder failed: %v", err)
		}
		if !shouldSync {
			t.Error("Expected ShouldSyncFolder to return true when cache is stale (older than 5 minutes)")
		}
	})
}

func getTestEncryptor(t *testing.T) *crypto.Encryptor {
	t.Helper()

	// Use the same test key pattern as api package tests
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, err := crypto.NewEncryptor(base64Key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	return encryptor
}

func TestGetFolderSyncInfoWithUID(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	// Ensure the folder_sync_timestamps table exists with new columns
	_, err := pool.Exec(ctx, `
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

	userID, err := db.GetOrCreateUser(ctx, pool, "uid-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	folderName := "INBOX"

	t.Run("GetFolderSyncInfo returns UID when set", func(t *testing.T) {
		lastUID := int64(50000)
		err := db.SetFolderSyncInfo(ctx, pool, userID, folderName, &lastUID)
		if err != nil {
			t.Fatalf("SetFolderSyncInfo failed: %v", err)
		}

		info, err := db.GetFolderSyncInfo(ctx, pool, userID, folderName)
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
	})

	t.Run("GetFolderSyncInfo returns nil UID when not set", func(t *testing.T) {
		err := db.SetFolderSyncInfo(ctx, pool, userID, "TestFolder", nil)
		if err != nil {
			t.Fatalf("SetFolderSyncInfo failed: %v", err)
		}

		info, err := db.GetFolderSyncInfo(ctx, pool, userID, "TestFolder")
		if err != nil {
			t.Fatalf("GetFolderSyncInfo failed: %v", err)
		}
		if info == nil {
			t.Fatal("Expected sync info, got nil")
		}
		if info.LastSyncedUID != nil {
			t.Errorf("Expected LastSyncedUID to be nil, got %d", *info.LastSyncedUID)
		}
	})
}

// Note: Full unit tests for SyncThreadsForFolder with mocks would require:
// 1. Creating interfaces for db operations and IMAP client pool
// 2. Refactoring Service to accept these interfaces
// 3. Creating mock implementations
//
// This is a larger refactoring. For now, integration tests (like above)
// verify the logic works correctly with a real database.
//
// To properly test SyncThreadsForFolder with mocks, we would need:
// - IMAPService interface with ShouldSyncFolder and SyncThreadsForFolder methods
// - DBService interface with GetUserSettings, SaveThread, SaveMessage, etc.
// - IMAPPool interface with GetClient method
// - Mock implementations of these interfaces
//
// The following functions would benefit from unit tests with mocks:
// - tryIncrementalSync: Requires mock IMAP client with UidSearch
// - performFullSync: Requires mock IMAP client with THREAD command
// - processIncrementalMessage: Can be tested with mock IMAP message
// - SearchUIDsSince: Requires mock IMAP client

func TestService_updateThreadCountInBackground(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	service := NewService(pool, encryptor)
	defer service.Close()

	ctx := context.Background()
	userID, err := db.GetOrCreateUser(ctx, pool, "thread-count-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	folderName := "INBOX"

	t.Run("handles database error gracefully", func(t *testing.T) {
		// Test that updateThreadCountInBackground handles database errors gracefully
		// by using an invalid userID that will cause UpdateThreadCount to fail
		// (it will try to update a non-existent folder_sync_timestamps row)
		invalidUserID := "00000000-0000-0000-0000-000000000000"

		// The function should log a warning but not crash
		service.updateThreadCountInBackground(invalidUserID, "NonExistentFolder")

		// Give the goroutine time to complete
		time.Sleep(200 * time.Millisecond)

		// Test should complete without panicking
		// If there's a panic, the test will fail
		// The function logs a warning for database errors, which is the expected behavior
	})

	t.Run("succeeds with valid database connection", func(t *testing.T) {
		// Test that the function works correctly with a valid connection
		service.updateThreadCountInBackground(userID, folderName)

		// Give the goroutine time to complete
		time.Sleep(100 * time.Millisecond)

		// Test should complete without panicking
		// If there's a panic, the test will fail
	})
}
