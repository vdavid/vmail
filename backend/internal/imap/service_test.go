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
		if err := db.SetFolderSyncTimestamp(ctx, pool, userID, folderName); err != nil {
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
