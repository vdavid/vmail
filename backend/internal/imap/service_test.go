package imap

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

// mockThreadCountUpdater is a mock implementation of ThreadCountUpdater for testing.
type mockThreadCountUpdater struct {
	mock.Mock
}

func (m *mockThreadCountUpdater) UpdateThreadCount(ctx context.Context, userID, folderName string) error {
	args := m.Called(ctx, userID, folderName)
	return args.Error(0)
}

// TestShouldSyncFolder tests the cache TTL logic using a real database.
// This is an integration test that verifies the ShouldSyncFolder logic works correctly.
func TestShouldSyncFolder(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

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

	encryptor := testutil.GetTestEncryptor(t)
	service := NewService(pool, NewPool(), encryptor)
	defer service.Close()

	userID, err := db.GetOrCreateUser(ctx, pool, "sync-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	folderName := "INBOX"

	tests := []struct {
		name        string
		setup       func()
		expected    bool
		description string
	}{
		{
			name:        "returns true when no sync timestamp exists",
			setup:       func() {}, // No setup needed
			expected:    true,
			description: "should sync when no timestamp exists",
		},
		{
			name: "returns false when cache is fresh",
			setup: func() {
				_ = db.SetFolderSyncInfo(ctx, pool, userID, folderName, nil)
			},
			expected:    false,
			description: "should not sync when cache is fresh",
		},
		{
			name: "returns true when cache is stale",
			setup: func() {
				_, _ = pool.Exec(ctx, `
					UPDATE folder_sync_timestamps 
					SET synced_at = $1 
					WHERE user_id = $2 AND folder_name = $3
				`, time.Now().Add(-10*time.Minute), userID, folderName)
			},
			expected:    true,
			description: "should sync when cache is stale (older than 5 minutes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			shouldSync, err := service.ShouldSyncFolder(ctx, userID, folderName)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, shouldSync, tt.description)
		})
	}
}

func TestGetFolderSyncInfoWithUID(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

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

	userID, err := db.GetOrCreateUser(ctx, pool, "uid-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	tests := []struct {
		name        string
		folderName  string
		setUID      *int64
		expectedUID *int64
		expectedNil bool
	}{
		{
			name:        "returns UID when set",
			folderName:  "INBOX",
			setUID:      intPtr(50000),
			expectedUID: intPtr(50000),
			expectedNil: false,
		},
		{
			name:        "returns nil UID when not set",
			folderName:  "TestFolder",
			setUID:      nil,
			expectedUID: nil,
			expectedNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.SetFolderSyncInfo(ctx, pool, userID, tt.folderName, tt.setUID)
			assert.NoError(t, err)

			info, err := db.GetFolderSyncInfo(ctx, pool, userID, tt.folderName)
			assert.NoError(t, err)
			assert.NotNil(t, info)

			if tt.expectedNil {
				assert.Nil(t, info.LastSyncedUID)
			} else {
				assert.NotNil(t, info.LastSyncedUID)
				assert.Equal(t, *tt.expectedUID, *info.LastSyncedUID)
			}
		})
	}
}

func TestService_updateThreadCountInBackground(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := testutil.GetTestEncryptor(t)
	mockUpdater := &mockThreadCountUpdater{}
	service := NewServiceWithThreadCountUpdater(pool, NewPool(), encryptor, mockUpdater)
	defer service.Close()

	ctx := context.Background()
	userID, err := db.GetOrCreateUser(ctx, pool, "thread-count-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	folderName := "INBOX"

	tests := []struct {
		name           string
		setupMock      func()
		testUserID     string
		testFolderName string
		expectError    bool
		waitTime       time.Duration
	}{
		{
			name: "succeeds with valid database connection",
			setupMock: func() {
				mockUpdater.On("UpdateThreadCount", mock.Anything, userID, folderName).
					Return(nil).
					Once()
			},
			testUserID:     userID,
			testFolderName: folderName,
			expectError:    false,
			waitTime:       100 * time.Millisecond,
		},
		{
			name: "handles database error gracefully",
			setupMock: func() {
				mockUpdater.On("UpdateThreadCount", mock.Anything, "00000000-0000-0000-0000-000000000000", "NonExistentFolder").
					Return(errors.New("database error")).
					Once()
			},
			testUserID:     "00000000-0000-0000-0000-000000000000",
			testFolderName: "NonExistentFolder",
			expectError:    true,
			waitTime:       200 * time.Millisecond,
		},
		{
			name: "handles context timeout",
			setupMock: func() {
				mockUpdater.On("UpdateThreadCount", mock.Anything, userID, folderName).
					Return(context.DeadlineExceeded).
					Once()
			},
			testUserID:     userID,
			testFolderName: folderName,
			expectError:    true,
			waitTime:       200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock for each test
			mockUpdater.ExpectedCalls = nil
			mockUpdater.Calls = nil
			tt.setupMock()

			service.updateThreadCountInBackground(tt.testUserID, tt.testFolderName)

			// Give the goroutine time to complete
			time.Sleep(tt.waitTime)

			// Verify mock expectations were met
			mockUpdater.AssertExpectations(t)

			// Test should complete without panicking
			// The function logs errors but doesn't panic, which is the expected behavior
		})
	}
}

func intPtr(i int64) *int64 {
	return &i
}

// TestCrossFolderThreading tests that messages across different folders (e.g., Inbox and Sent)
// are properly linked into the same thread when they reference each other via In-Reply-To headers.
//
// Note: The go-imap memory backend generates its own Message-ID headers for the envelope,
// so we need to sync messages first to discover their actual IDs, then create replies
// using those IDs.
func TestCrossFolderThreading(t *testing.T) {
	// Set test mode to use non-TLS connections
	if err := os.Setenv("VMAIL_TEST_MODE", "true"); err != nil {
		t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("VMAIL_TEST_MODE"); err != nil {
			t.Fatalf("Failed to unset VMAIL_TEST_MODE: %v", err)
		}
	}()

	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	encryptor := testutil.GetTestEncryptor(t)

	// Create test IMAP server
	imapServer := testutil.NewTestIMAPServer(t)
	defer imapServer.Close()

	// Ensure INBOX exists and create Sent folder
	imapServer.EnsureINBOX(t)
	client, cleanup := imapServer.Connect(t)
	err := client.Create("Sent")
	if err != nil && err.Error() != "Mailbox already exists" {
		t.Fatalf("Failed to create Sent folder: %v", err)
	}
	cleanup()

	// Create user with IMAP settings
	userID, err := db.GetOrCreateUser(ctx, pool, "threading-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	encryptedPassword, err := encryptor.Encrypt(imapServer.Password())
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	settings := &models.UserSettings{
		UserID:                   userID,
		IMAPServerHostname:       imapServer.Address,
		IMAPUsername:             imapServer.Username(),
		EncryptedIMAPPassword:    encryptedPassword,
		SMTPServerHostname:       "localhost:1025",
		SMTPUsername:             "test",
		EncryptedSMTPPassword:    encryptedPassword,
		UndoSendDelaySeconds:     20,
		PaginationThreadsPerPage: 100,
	}
	if err := db.SaveUserSettings(ctx, pool, settings); err != nil {
		t.Fatalf("Failed to save user settings: %v", err)
	}

	// Create IMAP service
	imapPool := NewPool()
	defer imapPool.Close()

	service := NewService(pool, imapPool, encryptor)
	defer service.Close()

	baseTime := time.Now().Add(-1 * time.Hour)

	// Step 1: Add original message to INBOX and sync to get its actual Message-ID
	imapServer.AddMessage(t, "INBOX", "<original@test>", "Test Conversation", "sender@example.com", "test@example.com", baseTime)

	err = service.SyncThreadsForFolder(ctx, userID, "INBOX")
	if err != nil {
		t.Fatalf("Failed to sync INBOX (step 1): %v", err)
	}

	// Get the actual Message-ID of the synced message from the database
	// Filter by subject to get our message, not the pre-existing one
	var originalMsgID string
	err = pool.QueryRow(ctx, `
		SELECT message_id_header FROM messages 
		WHERE user_id = $1 AND imap_folder_name = 'INBOX' AND subject = 'Test Conversation'
		LIMIT 1
	`, userID).Scan(&originalMsgID)
	if err != nil {
		t.Fatalf("Failed to get original message ID: %v", err)
	}
	t.Logf("Original message ID: %s", originalMsgID)

	// Step 2: Add reply to Sent folder with In-Reply-To pointing to original
	imapServer.AddMessageWithReplyTo(t, "Sent", "<reply@test>", "Re: Test Conversation", "test@example.com", "sender@example.com", originalMsgID, baseTime.Add(10*time.Minute))

	err = service.SyncThreadsForFolder(ctx, userID, "Sent")
	if err != nil {
		t.Fatalf("Failed to sync Sent folder: %v", err)
	}

	// Get the actual Message-ID of the reply from the database
	var replyMsgID string
	err = pool.QueryRow(ctx, `
		SELECT message_id_header FROM messages 
		WHERE user_id = $1 AND imap_folder_name = 'Sent'
		ORDER BY sent_at DESC NULLS LAST
		LIMIT 1
	`, userID).Scan(&replyMsgID)
	if err != nil {
		t.Fatalf("Failed to get reply message ID: %v", err)
	}
	t.Logf("Reply message ID: %s", replyMsgID)

	// Step 3: Add reply-to-reply to INBOX with In-Reply-To pointing to the reply
	imapServer.AddMessageWithReplyTo(t, "INBOX", "<reply-to-reply@test>", "Re: Re: Test Conversation", "sender@example.com", "test@example.com", replyMsgID, baseTime.Add(20*time.Minute))

	// Clear sync info to force re-sync
	_, _ = pool.Exec(ctx, `DELETE FROM folder_sync_timestamps WHERE user_id = $1`, userID)

	err = service.SyncThreadsForFolder(ctx, userID, "INBOX")
	if err != nil {
		t.Fatalf("Failed to sync INBOX (step 3): %v", err)
	}

	// Debug: List all messages with their thread IDs
	debugRows, err := pool.Query(ctx, `
		SELECT m.message_id_header, m.subject, m.imap_folder_name, t.stable_thread_id
		FROM messages m
		JOIN threads t ON m.thread_id = t.id
		WHERE m.user_id = $1
		ORDER BY m.sent_at NULLS LAST
	`, userID)
	if err != nil {
		t.Fatalf("Failed to query messages for debug: %v", err)
	}
	defer debugRows.Close()
	t.Log("All messages:")
	for debugRows.Next() {
		var msgID, subj, folder, stableID string
		if err := debugRows.Scan(&msgID, &subj, &folder, &stableID); err != nil {
			t.Logf("Failed to scan row: %v", err)
			continue
		}
		t.Logf("  Message-ID=%s, Subject=%s, Folder=%s, Thread=%s", msgID, subj, folder, stableID)
	}

	// Verify: All 3 messages should be in the same thread
	// Count threads that contain messages with "Test Conversation" subject
	var threadCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT t.id) 
		FROM threads t 
		INNER JOIN messages m ON t.id = m.thread_id
		WHERE t.user_id = $1 AND m.subject LIKE '%Test Conversation%'
	`, userID).Scan(&threadCount)
	if err != nil {
		t.Fatalf("Failed to count threads: %v", err)
	}

	// We expect exactly 1 thread containing all our conversation messages
	assert.Equal(t, 1, threadCount, "Expected exactly 1 thread for the conversation")

	// Get the thread that contains our messages
	var threadID string
	err = pool.QueryRow(ctx, `
		SELECT DISTINCT t.id 
		FROM threads t 
		INNER JOIN messages m ON t.id = m.thread_id
		WHERE t.user_id = $1
		AND m.subject LIKE '%Test Conversation%'
	`, userID).Scan(&threadID)
	if err != nil {
		t.Fatalf("Failed to get thread: %v", err)
	}

	messages, err := db.GetMessagesForThread(ctx, pool, threadID)
	if err != nil {
		t.Fatalf("Failed to get messages for thread: %v", err)
	}

	assert.Len(t, messages, 3, "Expected 3 messages in the thread")

	// Verify messages are from different folders
	folderCounts := make(map[string]int)
	for _, msg := range messages {
		folderCounts[msg.IMAPFolderName]++
	}
	assert.Equal(t, 2, folderCounts["INBOX"], "Expected 2 messages from INBOX")
	assert.Equal(t, 1, folderCounts["Sent"], "Expected 1 message from Sent")

	// Verify the thread's stable_thread_id is the original message's Message-ID
	var stableThreadID string
	err = pool.QueryRow(ctx, `SELECT stable_thread_id FROM threads WHERE id = $1`, threadID).Scan(&stableThreadID)
	if err != nil {
		t.Fatalf("Failed to get stable_thread_id: %v", err)
	}
	assert.Equal(t, originalMsgID, stableThreadID, "stable_thread_id should be the original message's Message-ID")
}

// TestThreadChainCompletion tests that syncing a folder with a message containing References
// will fetch all referenced messages from other folders (thread chain completion).
func TestThreadChainCompletion(t *testing.T) {
	// Set test mode to disable TLS for IMAP connections
	if err := os.Setenv("VMAIL_TEST_MODE", "true"); err != nil {
		t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("VMAIL_TEST_MODE"); err != nil {
			t.Fatalf("Failed to unset VMAIL_TEST_MODE: %v", err)
		}
	}()

	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	encryptor := testutil.GetTestEncryptor(t)

	// Start IMAP server
	imapServer := testutil.NewTestIMAPServer(t)
	defer imapServer.Close()

	// Ensure INBOX exists and create Sent folder
	imapServer.EnsureINBOX(t)
	client, cleanupClient := imapServer.Connect(t)
	err := client.Create("Sent")
	if err != nil && err.Error() != "Mailbox already exists" {
		t.Fatalf("Failed to create Sent folder: %v", err)
	}
	cleanupClient()

	// Create user with IMAP settings
	userID, err := db.GetOrCreateUser(ctx, pool, "chain-completion-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	encryptedPassword, err := encryptor.Encrypt(imapServer.Password())
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	settings := &models.UserSettings{
		UserID:                   userID,
		IMAPServerHostname:       imapServer.Address,
		IMAPUsername:             imapServer.Username(),
		EncryptedIMAPPassword:    encryptedPassword,
		SMTPServerHostname:       "localhost:1025",
		SMTPUsername:             "test",
		EncryptedSMTPPassword:    encryptedPassword,
		UndoSendDelaySeconds:     20,
		PaginationThreadsPerPage: 100,
	}
	if err := db.SaveUserSettings(ctx, pool, settings); err != nil {
		t.Fatalf("Failed to save user settings: %v", err)
	}

	// Create IMAP service
	imapPool := NewPool()
	defer imapPool.Close()

	service := NewService(pool, imapPool, encryptor)
	defer service.Close()

	baseTime := time.Now().Add(-1 * time.Hour)

	// Create a thread with 3 messages:
	// 1. Original in INBOX (root)
	// 2. Reply in Sent (references original)
	// 3. Reply-to-reply in INBOX (references both previous messages)

	originalMsgID := "<chain-original@test>"
	replyMsgID := "<chain-reply@test>"
	replyToReplyMsgID := "<chain-reply2@test>"

	// Add messages to IMAP with References headers
	imapServer.AddMessage(t, "INBOX", originalMsgID, "Chain Test", "sender@example.com", "test@example.com", baseTime)

	// Reply with References pointing to original
	imapServer.AddMessageWithReferences(t, "Sent", replyMsgID, "Re: Chain Test", "test@example.com", "sender@example.com",
		originalMsgID, []string{originalMsgID}, baseTime.Add(10*time.Minute))

	// Reply-to-reply with full References chain
	imapServer.AddMessageWithReferences(t, "INBOX", replyToReplyMsgID, "Re: Re: Chain Test", "sender@example.com", "test@example.com",
		replyMsgID, []string{originalMsgID, replyMsgID}, baseTime.Add(20*time.Minute))

	// KEY TEST: Only sync INBOX - the chain completion should fetch the Sent message
	err = service.SyncThreadsForFolder(ctx, userID, "INBOX")
	if err != nil {
		t.Fatalf("Failed to sync INBOX: %v", err)
	}

	// Debug: List all messages
	debugRows, err := pool.Query(ctx, `
		SELECT m.message_id_header, m.subject, m.imap_folder_name, t.stable_thread_id
		FROM messages m
		JOIN threads t ON m.thread_id = t.id
		WHERE m.user_id = $1
		ORDER BY m.sent_at NULLS LAST
	`, userID)
	if err != nil {
		t.Fatalf("Failed to query messages for debug: %v", err)
	}
	t.Log("All messages after syncing only INBOX:")
	for debugRows.Next() {
		var msgID, subj, folder, stableID string
		if err := debugRows.Scan(&msgID, &subj, &folder, &stableID); err != nil {
			t.Logf("Failed to scan row: %v", err)
			continue
		}
		t.Logf("  Message-ID=%s, Subject=%s, Folder=%s, Thread=%s", msgID, subj, folder, stableID)
	}
	if err := debugRows.Err(); err != nil {
		t.Fatalf("Error iterating debug rows: %v", err)
	}
	debugRows.Close()

	// Verify: All 3 messages should be synced (including the Sent message via chain completion)
	var messageCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM messages 
		WHERE user_id = $1 AND subject LIKE '%Chain Test%'
	`, userID).Scan(&messageCount)
	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}

	assert.Equal(t, 3, messageCount, "Expected 3 messages (chain completion should have fetched the Sent message)")

	// Verify all messages are in the same thread
	var threadCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT t.id) 
		FROM threads t 
		INNER JOIN messages m ON t.id = m.thread_id
		WHERE t.user_id = $1 AND m.subject LIKE '%Chain Test%'
	`, userID).Scan(&threadCount)
	if err != nil {
		t.Fatalf("Failed to count threads: %v", err)
	}

	assert.Equal(t, 1, threadCount, "Expected exactly 1 thread for the conversation")

	// Verify we have messages from both folders
	var sentCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM messages 
		WHERE user_id = $1 AND imap_folder_name = 'Sent' AND subject LIKE '%Chain Test%'
	`, userID).Scan(&sentCount)
	if err != nil {
		t.Fatalf("Failed to count Sent messages: %v", err)
	}

	assert.Equal(t, 1, sentCount, "Expected 1 message from Sent folder (fetched via chain completion)")
}

// TestChildMessageFetching tests that syncing a folder with a root message
// will automatically fetch child messages (replies) from other folders.
// This is the reverse of parent fetching - we fetch messages that REFERENCE the synced messages.
func TestChildMessageFetching(t *testing.T) {
	// Set test mode to disable TLS for IMAP connections
	if err := os.Setenv("VMAIL_TEST_MODE", "true"); err != nil {
		t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("VMAIL_TEST_MODE"); err != nil {
			t.Fatalf("Failed to unset VMAIL_TEST_MODE: %v", err)
		}
	}()

	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	encryptor := testutil.GetTestEncryptor(t)

	imapServer := testutil.NewTestIMAPServer(t)
	defer imapServer.Close()

	imapServer.EnsureINBOX(t)
	client, cleanupClient := imapServer.Connect(t)
	err := client.Create("Sent")
	if err != nil && err.Error() != "Mailbox already exists" {
		t.Fatalf("Failed to create Sent folder: %v", err)
	}
	cleanupClient()

	userID, err := db.GetOrCreateUser(ctx, pool, "child-fetch-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	encryptedPassword, err := encryptor.Encrypt(imapServer.Password())
	if err != nil {
		t.Fatalf("Failed to encrypt password: %v", err)
	}

	settings := &models.UserSettings{
		UserID:                   userID,
		IMAPServerHostname:       imapServer.Address,
		IMAPUsername:             imapServer.Username(),
		EncryptedIMAPPassword:    encryptedPassword,
		SMTPServerHostname:       "localhost:1025",
		SMTPUsername:             "test",
		EncryptedSMTPPassword:    encryptedPassword,
		UndoSendDelaySeconds:     20,
		PaginationThreadsPerPage: 100,
	}
	if err := db.SaveUserSettings(ctx, pool, settings); err != nil {
		t.Fatalf("Failed to save user settings: %v", err)
	}

	imapPool := NewPool()
	defer imapPool.Close()

	service := NewService(pool, imapPool, encryptor)
	defer service.Close()

	baseTime := time.Now().Add(-1 * time.Hour)

	// Create messages:
	// 1. Root message in INBOX (no References/In-Reply-To - this is the original email received)
	// 2. Reply in Sent (has In-Reply-To pointing to root)
	// When we sync INBOX, child fetching should find the reply in Sent

	rootMsgID := "<child-test-root@example.com>"
	replyMsgID := "<child-test-reply@example.com>"

	// Add root message to INBOX (no In-Reply-To, no References)
	imapServer.AddMessage(t, "INBOX", rootMsgID, "Original Email", "sender@example.com", "test@example.com", baseTime)

	// Add reply to Sent (In-Reply-To points to root)
	imapServer.AddMessageWithReplyTo(t, "Sent", replyMsgID, "Re: Original Email", "test@example.com", "sender@example.com",
		rootMsgID, baseTime.Add(10*time.Minute))

	// KEY TEST: Only sync INBOX - child fetching should fetch the Sent reply
	err = service.SyncThreadsForFolder(ctx, userID, "INBOX")
	if err != nil {
		t.Fatalf("Failed to sync INBOX: %v", err)
	}

	// Debug output
	debugRows, err := pool.Query(ctx, `
		SELECT m.message_id_header, m.subject, m.imap_folder_name, t.stable_thread_id
		FROM messages m
		JOIN threads t ON m.thread_id = t.id
		WHERE m.user_id = $1
		ORDER BY m.sent_at NULLS LAST
	`, userID)
	if err != nil {
		t.Fatalf("Failed to query messages for debug: %v", err)
	}
	t.Log("All messages after syncing only INBOX:")
	for debugRows.Next() {
		var msgID, subj, folder, stableID string
		if err := debugRows.Scan(&msgID, &subj, &folder, &stableID); err != nil {
			t.Logf("Failed to scan row: %v", err)
			continue
		}
		t.Logf("  Message-ID=%s, Subject=%s, Folder=%s, Thread=%s", msgID, subj, folder, stableID)
	}
	if err := debugRows.Err(); err != nil {
		t.Fatalf("Error iterating debug rows: %v", err)
	}
	debugRows.Close()

	// Verify: Both messages should be synced
	var messageCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM messages 
		WHERE user_id = $1 AND subject LIKE '%Original Email%'
	`, userID).Scan(&messageCount)
	if err != nil {
		t.Fatalf("Failed to count messages: %v", err)
	}

	assert.Equal(t, 2, messageCount, "Expected 2 messages (child fetching should have fetched the Sent reply)")

	// Verify both are in the same thread
	var threadCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT t.id) 
		FROM threads t 
		INNER JOIN messages m ON t.id = m.thread_id
		WHERE t.user_id = $1 AND m.subject LIKE '%Original Email%'
	`, userID).Scan(&threadCount)
	if err != nil {
		t.Fatalf("Failed to count threads: %v", err)
	}

	assert.Equal(t, 1, threadCount, "Expected exactly 1 thread for the conversation")

	// Verify we have the Sent message
	var sentCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM messages 
		WHERE user_id = $1 AND imap_folder_name = 'Sent' AND subject LIKE '%Original Email%'
	`, userID).Scan(&sentCount)
	if err != nil {
		t.Fatalf("Failed to count Sent messages: %v", err)
	}

	assert.Equal(t, 1, sentCount, "Expected 1 message from Sent folder (fetched via child fetching)")
}
