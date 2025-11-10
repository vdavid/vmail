package imap

import (
	"context"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestSearchUIDsSince(t *testing.T) {
	server := testutil.NewTestIMAPServer(t)
	defer server.Close()

	// Ensure INBOX exists
	server.EnsureINBOX(t)

	// Add test messages
	now := time.Now()
	uid1 := server.AddMessage(t, "INBOX", "<msg1@test>", "Subject 1", "from@test.com", "to@test.com", now.Add(-2*time.Hour))
	uid2 := server.AddMessage(t, "INBOX", "<msg2@test>", "Subject 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))
	uid3 := server.AddMessage(t, "INBOX", "<msg3@test>", "Subject 3", "from@test.com", "to@test.com", now)

	// Connect to get client for SearchUIDsSince
	client, clientCleanup := server.Connect(t)
	defer clientCleanup()

	// Select INBOX before searching
	_, err := client.Select("INBOX", false)
	if err != nil {
		t.Fatalf("Failed to select INBOX: %v", err)
	}

	t.Run("finds all UIDs when minUID is 1", func(t *testing.T) {
		uids, err := SearchUIDsSince(client, 1)
		if err != nil {
			t.Fatalf("SearchUIDsSince failed: %v", err)
		}
		// Memory backend creates a default message with UID 6, plus our three test messages
		// So we expect 4 UIDs total (6, 7, 8, 9)
		if len(uids) != 4 {
			t.Errorf("Expected 4 UIDs (1 default + 3 test), got %d: %v", len(uids), uids)
		}
	})

	t.Run("finds only UIDs >= minUID", func(t *testing.T) {
		// Search for UIDs >= uid2
		uids, err := SearchUIDsSince(client, uid2)
		if err != nil {
			t.Fatalf("SearchUIDsSince failed: %v", err)
		}
		if len(uids) != 2 {
			t.Errorf("Expected 2 UIDs (uid2 and uid3), got %d: %v", len(uids), uids)
		}
		// Check that uid1 is not included
		for _, uid := range uids {
			if uid == uid1 {
				t.Errorf("UID %d should not be included", uid1)
			}
		}
	})

	t.Run("returns empty when minUID is higher than all UIDs", func(t *testing.T) {
		uids, err := SearchUIDsSince(client, uid3+1)
		if err != nil {
			t.Fatalf("SearchUIDsSince failed: %v", err)
		}
		if len(uids) != 0 {
			t.Errorf("Expected 0 UIDs, got %d: %v", len(uids), uids)
		}
	})
}

func TestTryIncrementalSync(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	// Ensure folder_sync_timestamps table exists
	ctx := context.Background()
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

	// Setup test IMAP server
	server := testutil.NewTestIMAPServer(t)
	defer server.Close()

	// Ensure INBOX exists
	server.EnsureINBOX(t)

	// Connect to get client
	client, clientCleanup := server.Connect(t)
	defer clientCleanup()

	encryptor := getTestEncryptor(t)
	service := NewService(pool, encryptor)
	defer service.Close()

	userID, err := db.GetOrCreateUser(ctx, pool, "incremental-test@example.com")
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

	// Add initial messages
	now := time.Now()
	_ = server.AddMessage(t, folderName, "<initial1@test>", "Initial 1", "from@test.com", "to@test.com", now.Add(-2*time.Hour))
	uid2 := server.AddMessage(t, folderName, "<initial2@test>", "Initial 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))

	// Set sync info to uid2 (we've synced up to uid2)
	lastUID := int64(uid2)
	err = db.SetFolderSyncInfo(ctx, pool, userID, folderName, &lastUID)
	if err != nil {
		t.Fatalf("Failed to set folder sync info: %v", err)
	}

	// Get sync info
	syncInfo, err := db.GetFolderSyncInfo(ctx, pool, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to get folder sync info: %v", err)
	}

	t.Run("returns false when syncInfo is nil", func(t *testing.T) {
		result, ok := service.tryIncrementalSync(ctx, client, userID, folderName, nil)
		if ok {
			t.Error("Expected tryIncrementalSync to return false when syncInfo is nil")
		}
		if result.shouldReturn {
			t.Error("Expected shouldReturn to be false")
		}
	})

	t.Run("returns false when LastSyncedUID is nil", func(t *testing.T) {
		info := &db.FolderSyncInfo{LastSyncedUID: nil}
		result, ok := service.tryIncrementalSync(ctx, client, userID, folderName, info)
		if ok {
			t.Error("Expected tryIncrementalSync to return false when LastSyncedUID is nil")
		}
		if result.shouldReturn {
			t.Error("Expected shouldReturn to be false")
		}
	})

	t.Run("finds new messages after last synced UID", func(t *testing.T) {
		// Add a new message
		uid3 := server.AddMessage(t, folderName, "<new1@test>", "New Message", "from@test.com", "to@test.com", now)

		// Reconnect to get the fresh client (needed for memory backend)
		clientCleanup()
		client, clientCleanup = server.Connect(t)
		defer clientCleanup()
		_, _ = client.Select(folderName, false)

		result, ok := service.tryIncrementalSync(ctx, client, userID, folderName, syncInfo)
		if !ok {
			t.Error("Expected tryIncrementalSync to return true")
		}
		if result.shouldReturn {
			t.Error("Expected shouldReturn to be false (there are new messages)")
		}
		if len(result.uidsToSync) != 1 {
			t.Errorf("Expected 1 new UID, got %d", len(result.uidsToSync))
		}
		if result.uidsToSync[0] != uid3 {
			t.Errorf("Expected UID %d, got %d", uid3, result.uidsToSync[0])
		}
		if result.highestUID != uid3 {
			t.Errorf("Expected highest UID %d, got %d", uid3, result.highestUID)
		}
	})

	t.Run("returns shouldReturn=true when no new messages", func(t *testing.T) {
		// Reconnect
		clientCleanup()
		client, clientCleanup = server.Connect(t)
		defer clientCleanup()
		_, _ = client.Select(folderName, false)

		// Get the highest UID from the server
		criteria := imap.NewSearchCriteria()
		allUIDs, err := client.UidSearch(criteria)
		if err != nil {
			t.Fatalf("Failed to search for UIDs: %v", err)
		}
		if len(allUIDs) == 0 {
			t.Fatal("No UIDs found")
		}
		highestUID := allUIDs[len(allUIDs)-1]

		// Update sync info to the latest UID
		latestUID := int64(highestUID)
		err = db.SetFolderSyncInfo(ctx, pool, userID, folderName, &latestUID)
		if err != nil {
			t.Fatalf("Failed to update sync info: %v", err)
		}

		updatedSyncInfo, err := db.GetFolderSyncInfo(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("Failed to get updated sync info: %v", err)
		}

		result, ok := service.tryIncrementalSync(ctx, client, userID, folderName, updatedSyncInfo)
		if !ok {
			t.Error("Expected tryIncrementalSync to return true")
		}
		if !result.shouldReturn {
			t.Error("Expected shouldReturn to be true (no new messages)")
		}
		if len(result.uidsToSync) != 0 {
			t.Errorf("Expected 0 UIDs to sync, got %d", len(result.uidsToSync))
		}
	})
}

func TestProcessIncrementalMessage(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	// Ensure folder_sync_timestamps table exists
	ctx := context.Background()
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

	encryptor := getTestEncryptor(t)
	service := NewService(pool, encryptor)
	defer service.Close()

	userID, err := db.GetOrCreateUser(ctx, pool, "process-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	folderName := "INBOX"

	t.Run("creates new thread for new message", func(t *testing.T) {
		// Create a test IMAP message
		messageID := "<new-thread@test>"
		subject := "New Thread Subject"
		now := time.Now()

		imapMsg := &imap.Message{
			Uid: 1,
			Envelope: &imap.Envelope{
				MessageId: messageID,
				Subject:   subject,
				Date:      now,
				From: []*imap.Address{
					{MailboxName: "from", HostName: "test.com"},
				},
				To: []*imap.Address{
					{MailboxName: "to", HostName: "test.com"},
				},
			},
			Flags: []string{imap.SeenFlag},
		}

		err := service.processIncrementalMessage(ctx, imapMsg, userID, folderName)
		if err != nil {
			t.Fatalf("processIncrementalMessage failed: %v", err)
		}

		// Verify thread was created
		thread, err := db.GetThreadByStableID(ctx, pool, userID, messageID)
		if err != nil {
			t.Fatalf("Failed to get thread: %v", err)
		}
		if thread.Subject != subject {
			t.Errorf("Expected subject %s, got %s", subject, thread.Subject)
		}

		// Verify the message was saved
		msg, err := db.GetMessageByMessageID(ctx, pool, userID, messageID)
		if err != nil {
			t.Fatalf("Failed to get message: %v", err)
		}
		if msg.ThreadID != thread.ID {
			t.Errorf("Message thread ID doesn't match: expected %s, got %s", thread.ID, msg.ThreadID)
		}
	})

	t.Run("uses existing thread when message already exists", func(t *testing.T) {
		// Create a thread and message first
		messageID := "<existing@test>"
		thread := &models.Thread{
			UserID:         userID,
			StableThreadID: messageID,
			Subject:        "Existing Thread",
		}
		err := db.SaveThread(ctx, pool, thread)
		if err != nil {
			t.Fatalf("Failed to save thread: %v", err)
		}

		// Create an IMAP message with the same Message-ID
		imapMsg := &imap.Message{
			Uid: 2,
			Envelope: &imap.Envelope{
				MessageId: messageID,
				Subject:   "Existing Thread",
				Date:      time.Now(),
				From: []*imap.Address{
					{MailboxName: "from", HostName: "test.com"},
				},
				To: []*imap.Address{
					{MailboxName: "to", HostName: "test.com"},
				},
			},
			Flags: []string{imap.SeenFlag},
		}

		err = service.processIncrementalMessage(ctx, imapMsg, userID, folderName)
		if err != nil {
			t.Fatalf("processIncrementalMessage failed: %v", err)
		}

		// Verify message was added to the existing thread
		msg, err := db.GetMessageByMessageID(ctx, pool, userID, messageID)
		if err != nil {
			t.Fatalf("Failed to get message: %v", err)
		}
		if msg.ThreadID != thread.ID {
			t.Errorf("Message should be in existing thread %s, got %s", thread.ID, msg.ThreadID)
		}
	})

	t.Run("skips message without Message-ID", func(t *testing.T) {
		imapMsg := &imap.Message{
			Uid: 3,
			Envelope: &imap.Envelope{
				// No Message-ID
				Subject: "No Message-ID",
				Date:    time.Now(),
			},
			Flags: []string{imap.SeenFlag},
		}

		err := service.processIncrementalMessage(ctx, imapMsg, userID, folderName)
		if err != nil {
			t.Fatalf("processIncrementalMessage should not fail for message without Message-ID: %v", err)
		}

		// Message should not be saved
		// (We can't easily check this without querying, but the function should return nil)
	})
}
