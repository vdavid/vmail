package imap

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

// TestSyncThreadsForFolder_DetectsNewEmail tests the full flow:
// 1. Add initial emails to IMAP and sync them
// 2. List emails from database (get initial count)
// 3. Add a new email to IMAP server
// 4. Sync again (simulating what IDLE listener would do)
// 5. List emails again and verify the new email appears
func TestSyncThreadsForFolder_DetectsNewEmail(t *testing.T) {
	// Set test mode to disable TLS for test IMAP server
	err := os.Setenv("VMAIL_TEST_MODE", "true")
	if err != nil {
		t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
	}
	defer func() {
		err := os.Unsetenv("VMAIL_TEST_MODE")
		if err != nil {
			t.Fatalf("Failed to unset VMAIL_TEST_MODE: %v", err)
		}
	}()

	pool := testutil.NewTestDB(t)
	defer pool.Close()

	// Setup test IMAP server
	server := testutil.NewTestIMAPServer(t)
	defer server.Close()

	// Ensure INBOX exists
	server.EnsureINBOX(t)

	encryptor := getTestEncryptor(t)
	service := NewService(pool, NewPool(), encryptor)
	defer service.Close()

	ctx := context.Background()
	userID, err := db.GetOrCreateUser(ctx, pool, "sync-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Save user settings with the test IMAP server
	password := server.Password()
	encryptedPassword, err := encryptor.Encrypt(password)
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	// Also encrypt SMTP password (required field)
	encryptedSMTPPassword, err := encryptor.Encrypt(password)
	if err != nil {
		t.Fatalf("Failed to encrypt SMTP password: %v", err)
	}

	settings := &models.UserSettings{
		UserID:                userID,
		IMAPServerHostname:    server.Address,
		IMAPUsername:          server.Username(),
		EncryptedIMAPPassword: encryptedPassword,
		EncryptedSMTPPassword: encryptedSMTPPassword,
	}
	err = db.SaveUserSettings(ctx, pool, settings)
	if err != nil {
		t.Fatalf("Failed to save user settings: %v", err)
	}

	folderName := "INBOX"

	// Step 1: Add initial emails to IMAP
	now := time.Now()
	initialMessageID1 := "<initial1@test>"
	initialMessageID2 := "<initial2@test>"
	_ = server.AddMessage(t, folderName, initialMessageID1, "Initial Email 1", "from1@test.com", "to@test.com", now.Add(-2*time.Hour))
	_ = server.AddMessage(t, folderName, initialMessageID2, "Initial Email 2", "from2@test.com", "to@test.com", now.Add(-1*time.Hour))

	// Step 2: Sync initial emails
	err = service.SyncThreadsForFolder(ctx, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to sync initial emails: %v", err)
	}

	// Step 3: List emails from database (get initial count)
	initialThreads, err := db.GetThreadsForFolder(ctx, pool, userID, folderName, 100, 0)
	if err != nil {
		t.Fatalf("Failed to get initial threads: %v", err)
	}
	initialCount := len(initialThreads)
	t.Logf("Initial thread count: %d", initialCount)

	// Verify we have the initial emails
	if initialCount < 2 {
		t.Errorf("Expected at least 2 initial threads, got %d", initialCount)
	}

	// Verify the initial emails are in the database
	foundInitial1 := false
	foundInitial2 := false
	for _, thread := range initialThreads {
		if thread.StableThreadID == initialMessageID1 {
			foundInitial1 = true
		}
		if thread.StableThreadID == initialMessageID2 {
			foundInitial2 = true
		}
	}
	if !foundInitial1 {
		t.Error("Initial email 1 not found in database")
	}
	if !foundInitial2 {
		t.Error("Initial email 2 not found in database")
	}

	// Step 4: Add a new email to IMAP server
	newMessageID := "<new-email@test>"
	newSubject := "New Email Subject"
	_ = server.AddMessage(t, folderName, newMessageID, newSubject, "newfrom@test.com", "to@test.com", now)

	// Step 5: Sync again (simulating what IDLE listener would do)
	err = service.SyncThreadsForFolder(ctx, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to sync new email: %v", err)
	}

	// Step 6: List emails again and verify the new email appears
	updatedThreads, err := db.GetThreadsForFolder(ctx, pool, userID, folderName, 100, 0)
	if err != nil {
		t.Fatalf("Failed to get updated threads: %v", err)
	}
	updatedCount := len(updatedThreads)
	t.Logf("Updated thread count: %d", updatedCount)

	// Verify the count increased
	if updatedCount <= initialCount {
		t.Errorf("Expected thread count to increase, got %d (was %d)", updatedCount, initialCount)
	}

	// Verify the new email is in the database
	foundNewEmail := false
	for _, thread := range updatedThreads {
		if thread.StableThreadID == newMessageID {
			foundNewEmail = true
			if thread.Subject != newSubject {
				t.Errorf("Expected subject %s, got %s", newSubject, thread.Subject)
			}
			break
		}
	}
	if !foundNewEmail {
		t.Errorf("New email with Message-ID %s not found in database. Threads: %v", newMessageID, getThreadIDs(updatedThreads))
	}

	// Verify the message was saved correctly
	msg, err := db.GetMessageByMessageID(ctx, pool, userID, newMessageID)
	if err != nil {
		t.Fatalf("Failed to get new message from database: %v", err)
	}
	if msg.Subject != newSubject {
		t.Errorf("Expected message subject %s, got %s", newSubject, msg.Subject)
	}
	if msg.IMAPFolderName != folderName {
		t.Errorf("Expected folder %s, got %s", folderName, msg.IMAPFolderName)
	}
}

// TestSyncThreadsForFolder_IncrementalSync verifies that incremental sync
// correctly picks up new emails after the last synced UID.
func TestSyncThreadsForFolder_IncrementalSync(t *testing.T) {
	// Set test mode to disable TLS for test IMAP server
	err := os.Setenv("VMAIL_TEST_MODE", "true")
	if err != nil {
		t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
	}
	defer func() {
		err := os.Unsetenv("VMAIL_TEST_MODE")
		if err != nil {
			t.Fatalf("Failed to unset VMAIL_TEST_MODE: %v", err)
		}
	}()

	pool := testutil.NewTestDB(t)
	defer pool.Close()

	// Ensure folder_sync_timestamps table exists
	ctx := context.Background()
	_, err2 := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS folder_sync_timestamps (
			user_id        UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			folder_name    TEXT        NOT NULL,
			synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_synced_uid BIGINT,
			thread_count   INT DEFAULT 0,
			PRIMARY KEY (user_id, folder_name)
		)
	`)
	if err2 != nil {
		t.Fatalf("Failed to ensure folder_sync_timestamps table exists: %v", err2)
	}

	// Setup test IMAP server
	server := testutil.NewTestIMAPServer(t)
	defer server.Close()

	// Ensure INBOX exists
	server.EnsureINBOX(t)

	encryptor := getTestEncryptor(t)
	service := NewService(pool, NewPool(), encryptor)
	defer service.Close()

	userID, err := db.GetOrCreateUser(ctx, pool, "incremental-sync-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Save user settings with the test IMAP server
	password := server.Password()
	encryptedPassword, err := encryptor.Encrypt(password)
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	encryptedSMTPPassword, err := encryptor.Encrypt(password)
	if err != nil {
		t.Fatalf("Failed to encrypt SMTP password: %v", err)
	}

	settings := &models.UserSettings{
		UserID:                userID,
		IMAPServerHostname:    server.Address,
		IMAPUsername:          server.Username(),
		EncryptedIMAPPassword: encryptedPassword,
		EncryptedSMTPPassword: encryptedSMTPPassword,
	}
	err = db.SaveUserSettings(ctx, pool, settings)
	if err != nil {
		t.Fatalf("Failed to save user settings: %v", err)
	}

	folderName := "INBOX"

	// Add initial messages and sync
	now := time.Now()
	_ = server.AddMessage(t, folderName, "<initial1@test>", "Initial 1", "from@test.com", "to@test.com", now.Add(-2*time.Hour))
	uid2 := server.AddMessage(t, folderName, "<initial2@test>", "Initial 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))

	// Sync initial messages
	err = service.SyncThreadsForFolder(ctx, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to sync initial messages: %v", err)
	}

	// Set sync info to uid2 (we've synced up to uid2)
	lastUID := int64(uid2)
	err = db.SetFolderSyncInfo(ctx, pool, userID, folderName, &lastUID)
	if err != nil {
		t.Fatalf("Failed to set folder sync info: %v", err)
	}

	// Add a new message after uid2
	newUID := server.AddMessage(t, folderName, "<new@test>", "New Message", "from@test.com", "to@test.com", now)

	// Sync again - should use incremental sync
	err = service.SyncThreadsForFolder(ctx, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to sync new message: %v", err)
	}

	// Verify the new message is in the database
	msg, err := db.GetMessageByMessageID(ctx, pool, userID, "<new@test>")
	if err != nil {
		t.Fatalf("Failed to get new message: %v", err)
	}

	if msg.IMAPUID != int64(newUID) {
		t.Errorf("Expected UID %d, got %d", newUID, msg.IMAPUID)
	}

	// Verify sync info was updated
	syncInfo, err := db.GetFolderSyncInfo(ctx, pool, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to get sync info: %v", err)
	}
	if syncInfo == nil {
		t.Fatal("Expected sync info to exist after sync")
	}

	if syncInfo.LastSyncedUID == nil || *syncInfo.LastSyncedUID != int64(newUID) {
		t.Errorf("Expected LastSyncedUID to be %d, got %v", newUID, syncInfo.LastSyncedUID)
	}
}

// TestSyncThreadsForFolder_CatchesUpOnMissedEmails tests that when syncing
// after a period of inactivity, all missed emails are synced.
// This simulates the scenario where emails arrive while WebSocket is disconnected,
// and then WebSocket connects and triggers a sync.
func TestSyncThreadsForFolder_CatchesUpOnMissedEmails(t *testing.T) {
	// Set test mode to disable TLS for test IMAP server
	err := os.Setenv("VMAIL_TEST_MODE", "true")
	if err != nil {
		t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
	}
	defer func() {
		err := os.Unsetenv("VMAIL_TEST_MODE")
		if err != nil {
			t.Fatalf("Failed to unset VMAIL_TEST_MODE: %v", err)
		}
	}()

	pool := testutil.NewTestDB(t)
	defer pool.Close()

	server := testutil.NewTestIMAPServer(t)
	defer server.Close()
	server.EnsureINBOX(t)

	encryptor := getTestEncryptor(t)
	service := NewService(pool, NewPool(), encryptor)
	defer service.Close()

	ctx := context.Background()
	userID, err := db.GetOrCreateUser(ctx, pool, "catchup-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Save user settings
	password := server.Password()
	encryptedPassword, _ := encryptor.Encrypt(password)
	encryptedSMTPPassword, _ := encryptor.Encrypt(password)
	settings := &models.UserSettings{
		UserID:                userID,
		IMAPServerHostname:    server.Address,
		IMAPUsername:          server.Username(),
		EncryptedIMAPPassword: encryptedPassword,
		EncryptedSMTPPassword: encryptedSMTPPassword,
	}
	_ = db.SaveUserSettings(ctx, pool, settings)

	folderName := "INBOX"
	now := time.Now()

	// Initial sync: add and sync some emails
	uid1 := server.AddMessage(t, folderName, "<initial1@test>", "Initial 1", "from@test.com", "to@test.com", now.Add(-2*time.Hour))
	_ = server.AddMessage(t, folderName, "<initial2@test>", "Initial 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))

	err = service.SyncThreadsForFolder(ctx, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to sync initial emails: %v", err)
	}

	// Set sync info to uid1 (simulating that we've synced up to uid1)
	// This simulates a previous sync that happened before WebSocket disconnected
	lastUID := int64(uid1)
	err = db.SetFolderSyncInfo(ctx, pool, userID, folderName, &lastUID)
	if err != nil {
		t.Fatalf("Failed to set folder sync info: %v", err)
	}

	// Simulate emails arriving while WebSocket was disconnected
	missedMessageID1 := "<missed1@test>"
	missedMessageID2 := "<missed2@test>"
	_ = server.AddMessage(t, folderName, missedMessageID1, "Missed Email 1", "from@test.com", "to@test.com", now.Add(-30*time.Minute))
	_ = server.AddMessage(t, folderName, missedMessageID2, "Missed Email 2", "from@test.com", "to@test.com", now)

	// Now simulate WebSocket connecting and triggering sync
	// This should catch up on all missed emails using incremental sync
	err = service.SyncThreadsForFolder(ctx, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to sync missed emails: %v", err)
	}

	// Verify both missed emails are in the database
	msg1, err := db.GetMessageByMessageID(ctx, pool, userID, missedMessageID1)
	if err != nil {
		t.Fatalf("Missed email 1 not found in database: %v", err)
	}
	if msg1.Subject != "Missed Email 1" {
		t.Errorf("Expected subject 'Missed Email 1', got %s", msg1.Subject)
	}

	msg2, err := db.GetMessageByMessageID(ctx, pool, userID, missedMessageID2)
	if err != nil {
		t.Fatalf("Missed email 2 not found in database: %v", err)
	}
	if msg2.Subject != "Missed Email 2" {
		t.Errorf("Expected subject 'Missed Email 2', got %s", msg2.Subject)
	}

	// Verify sync info was updated to the highest UID
	syncInfo, err := db.GetFolderSyncInfo(ctx, pool, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to get sync info: %v", err)
	}
	if syncInfo == nil {
		t.Fatal("Expected sync info to exist after sync")
	}
	if syncInfo.LastSyncedUID == nil {
		t.Error("Expected LastSyncedUID to be set after sync")
	}
}

// getThreadIDs is a helper to extract thread IDs for debugging.
func getThreadIDs(threads []*models.Thread) []string {
	ids := make([]string, len(threads))
	for i, thread := range threads {
		ids[i] = thread.StableThreadID
	}
	return ids
}
