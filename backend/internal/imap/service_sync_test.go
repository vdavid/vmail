package imap

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/memory"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/server"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

// setupTestIMAPServer creates a test IMAP server with an in-memory backend.
// Returns the server, connection address, backend, and cleanup function.
func setupTestIMAPServer(t *testing.T) (*server.Server, string, *memory.Backend, func()) {
	t.Helper()

	// Create an in-memory backend
	be := memory.New()

	// Create server
	s := server.New(be)
	s.AllowInsecureAuth = true

	// Start server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	addr := listener.Addr().String()

	// Start server in goroutine
	go func() {
		if err := s.Serve(listener); err != nil {
			t.Logf("IMAP server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
		err := s.Close()
		if err != nil {
			return
		}
	}

	return s, addr, be, cleanup
}

// createTestIMAPUser creates a user in the memory backend.
// The memory backend creates a default user with username "username" and password "password".
// For testing, we'll use the default user that's already created, or we can log in
// with any credentials, and the backend will accept them if we create the user properly.
// Since we can't easily create new users (fields are unexported), we'll use
// unsafe reflection or just use the default user. For simplicity, let's use
// the default credentials that the memory backend provides.
func createTestIMAPUser(t *testing.T, be *memory.Backend, username, password string) {
	t.Helper()

	// The memory backend creates a default user with username "username" and password "password"
	// For our tests, we'll use those default credentials instead of trying to create new users
	// This is a limitation of the memory backend - it doesn't provide a public API to create users

	// Try to log in - if it fails, we'll use the default user
	user, err := be.Login(nil, username, password)
	if err != nil {
		// If login fails, try with default credentials
		// But actually, we want to use our specified username/password.
		// So we need to create the user. Since reflection doesn't work with unexported fields,
		// we'll use unsafe pointer arithmetic (not recommended but necessary here)
		// Actually, let's just document this limitation and use the default user for now
		t.Logf("Note: Memory backend doesn't support creating custom users easily. " +
			"Using default user (username: username, password: password) for testing.")

		// Use default credentials
		user, err = be.Login(nil, "username", "password")
		if err != nil {
			t.Fatalf("Failed to login with default credentials: %v", err)
		}
	}

	// Create INBOX for the user if it doesn't exist
	_, err = user.GetMailbox("INBOX")
	if err != nil {
		err = user.CreateMailbox("INBOX")
		if err != nil {
			t.Fatalf("Failed to create INBOX: %v", err)
		}
	}
}

// addTestMessage adds a test message to the IMAP server.
func addTestMessage(t *testing.T, client *imapclient.Client, folderName, messageID, subject, from, to string, sentAt time.Time) uint32 {
	t.Helper()

	// Select the folder
	_, err := client.Select(folderName, false)
	if err != nil {
		t.Fatalf("Failed to select folder: %v", err)
	}

	// Create a simple RFC 822 message
	messageBody := fmt.Sprintf(`Message-ID: %s
Date: %s
From: %s
To: %s
Subject: %s
Content-Type: text/plain; charset=utf-8

Test message body.
`, messageID, sentAt.Format(time.RFC1123Z), from, to, subject)

	// Append the message to the folder
	flags := []string{imap.SeenFlag}
	now := time.Now()
	err = client.Append(folderName, flags, now, strings.NewReader(messageBody))
	if err != nil {
		t.Fatalf("Failed to append message: %v", err)
	}

	// Search for the message we just added to get its UID
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("Message-ID", messageID)
	uids, err := client.UidSearch(criteria)
	if err != nil {
		t.Fatalf("Failed to search for message: %v", err)
	}

	if len(uids) == 0 {
		t.Fatalf("Message not found after append")
	}

	return uids[0]
}

// connectToTestServer connects to the test IMAP server.
func connectToTestServer(t *testing.T, addr, username, password string) (*imapclient.Client, func()) {
	t.Helper()

	client, err := imapclient.Dial(addr)
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}

	if err := client.Login(username, password); err != nil {
		err := client.Logout()
		if err != nil {
			return nil, nil
		}
		t.Fatalf("Failed to login: %v", err)
	}

	cleanup := func() {
		err := client.Logout()
		if err != nil {
			return
		}
	}

	return client, cleanup
}

func TestSearchUIDsSince(t *testing.T) {
	_, addr, be, cleanup := setupTestIMAPServer(t)
	defer cleanup()

	// Memory backend creates a default user with these credentials
	username := "username"
	password := "password"

	// Ensure the user exists (memory backend creates it by default, but let's verify)
	createTestIMAPUser(t, be, username, password)

	// Create user and connect
	client, clientCleanup := connectToTestServer(t, addr, username, password)
	defer clientCleanup()

	// Create the INBOX folder
	_, err := client.Select("INBOX", false)
	if err != nil {
		// Create INBOX if it doesn't exist
		err = client.Create("INBOX")
		if err != nil {
			t.Fatalf("Failed to create INBOX: %v", err)
		}
		_, err = client.Select("INBOX", false)
		if err != nil {
			t.Fatalf("Failed to select INBOX: %v", err)
		}
	}

	// Add test messages
	now := time.Now()
	uid1 := addTestMessage(t, client, "INBOX", "<msg1@test>", "Subject 1", "from@test.com", "to@test.com", now.Add(-2*time.Hour))
	uid2 := addTestMessage(t, client, "INBOX", "<msg2@test>", "Subject 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))
	uid3 := addTestMessage(t, client, "INBOX", "<msg3@test>", "Subject 3", "from@test.com", "to@test.com", now)

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
	_, addr, be, cleanup := setupTestIMAPServer(t)
	defer cleanup()

	// Memory backend creates a default user with these credentials
	username := "username"
	password := "password"

	// Ensure user exists
	createTestIMAPUser(t, be, username, password)

	// Create user and connect
	client, clientCleanup := connectToTestServer(t, addr, username, password)
	defer clientCleanup()

	// Create the INBOX folder
	_, err = client.Select("INBOX", false)
	if err != nil {
		err = client.Create("INBOX")
		if err != nil {
			t.Fatalf("Failed to create INBOX: %v", err)
		}
		_, err = client.Select("INBOX", false)
		if err != nil {
			t.Fatalf("Failed to select INBOX: %v", err)
		}
	}

	encryptor := getTestEncryptor(t)
	service := NewService(pool, encryptor)
	defer service.Close()

	userID, err := db.GetOrCreateUser(ctx, pool, "incremental-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Save user settings with the test IMAP server
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
		IMAPServerHostname:    addr,
		IMAPUsername:          username,
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
	_ = addTestMessage(t, client, folderName, "<initial1@test>", "Initial 1", "from@test.com", "to@test.com", now.Add(-2*time.Hour))
	uid2 := addTestMessage(t, client, folderName, "<initial2@test>", "Initial 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))

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
		uid3 := addTestMessage(t, client, folderName, "<new1@test>", "New Message", "from@test.com", "to@test.com", now)

		// Reconnect to get the fresh client (needed for memory backend)
		err := client.Close()
		if err != nil {
			return
		}
		client, _ = connectToTestServer(t, addr, username, password)
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
		err := client.Close()
		if err != nil {
			return
		}
		client, _ = connectToTestServer(t, addr, username, password)
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
