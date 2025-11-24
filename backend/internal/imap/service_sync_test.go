package imap

import (
	"context"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestSearchUIDsSince(t *testing.T) {
	server := testutil.NewTestIMAPServer(t)
	defer server.Close()

	server.EnsureINBOX(t)

	now := time.Now()
	uid1 := server.AddMessage(t, "INBOX", "<msg1@test>", "Subject 1", "from@test.com", "to@test.com", now.Add(-2*time.Hour))
	uid2 := server.AddMessage(t, "INBOX", "<msg2@test>", "Subject 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))
	uid3 := server.AddMessage(t, "INBOX", "<msg3@test>", "Subject 3", "from@test.com", "to@test.com", now)

	client, clientCleanup := server.Connect(t)
	defer clientCleanup()

	_, err := client.Select("INBOX", false)
	if err != nil {
		t.Fatalf("Failed to select INBOX: %v", err)
	}

	tests := []struct {
		name        string
		minUID      uint32
		expectedLen int
		checkResult func(*testing.T, []uint32, uint32)
	}{
		{
			name:        "finds all UIDs when minUID is 1",
			minUID:      1,
			expectedLen: 4, // Memory backend creates a default message with UID 6, plus our three test messages
			checkResult: nil,
		},
		{
			name:        "finds only UIDs >= minUID",
			minUID:      uid2,
			expectedLen: 2,
			checkResult: func(t *testing.T, uids []uint32, uid1 uint32) {
				for _, uid := range uids {
					assert.NotEqual(t, uid1, uid, "uid1 should not be included")
				}
			},
		},
		{
			name:        "returns empty when minUID is higher than all UIDs",
			minUID:      uid3 + 1,
			expectedLen: 0,
			checkResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uids, err := SearchUIDsSince(client, tt.minUID)
			assert.NoError(t, err)
			assert.Len(t, uids, tt.expectedLen)
			if tt.checkResult != nil {
				tt.checkResult(t, uids, uid1)
			}
		})
	}
}

func TestTryIncrementalSync(t *testing.T) {
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

	server := testutil.NewTestIMAPServer(t)
	defer server.Close()
	server.EnsureINBOX(t)

	client, clientCleanup := server.Connect(t)
	defer clientCleanup()

	encryptor := testutil.GetTestEncryptor(t)
	service := NewService(pool, NewPool(), encryptor)
	defer service.Close()

	userID, err := db.GetOrCreateUser(ctx, pool, "incremental-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

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
	now := time.Now()
	_ = server.AddMessage(t, folderName, "<initial1@test>", "Initial 1", "from@test.com", "to@test.com", now.Add(-2*time.Hour))
	uid2 := server.AddMessage(t, folderName, "<initial2@test>", "Initial 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))

	lastUID := int64(uid2)
	err = db.SetFolderSyncInfo(ctx, pool, userID, folderName, &lastUID)
	if err != nil {
		t.Fatalf("Failed to set folder sync info: %v", err)
	}

	syncInfo, err := db.GetFolderSyncInfo(ctx, pool, userID, folderName)
	if err != nil {
		t.Fatalf("Failed to get folder sync info: %v", err)
	}

	t.Run("returns false when syncInfo is nil", func(t *testing.T) {
		result, ok := service.tryIncrementalSync(ctx, client, userID, folderName, nil)
		assert.False(t, ok)
		assert.False(t, result.shouldReturn)
	})

	t.Run("returns false when LastSyncedUID is nil", func(t *testing.T) {
		info := &db.FolderSyncInfo{LastSyncedUID: nil}
		result, ok := service.tryIncrementalSync(ctx, client, userID, folderName, info)
		assert.False(t, ok)
		assert.False(t, result.shouldReturn)
	})

	t.Run("finds new messages after last synced UID", func(t *testing.T) {
		uid3 := server.AddMessage(t, folderName, "<new1@test>", "New Message", "from@test.com", "to@test.com", now)

		clientCleanup()
		c, cleanup := server.Connect(t)
		defer cleanup()
		_, _ = c.Select(folderName, false)

		result, ok := service.tryIncrementalSync(ctx, c, userID, folderName, syncInfo)
		assert.True(t, ok)
		assert.False(t, result.shouldReturn, "shouldReturn should be false when there are new messages")
		assert.Len(t, result.uidsToSync, 1)
		assert.Equal(t, uid3, result.uidsToSync[0])
		assert.Equal(t, uid3, result.highestUID)
	})

	t.Run("returns shouldReturn=true when no new messages", func(t *testing.T) {
		clientCleanup()
		c, cleanup := server.Connect(t)
		defer cleanup()
		_, _ = c.Select(folderName, false)

		criteria := imap.NewSearchCriteria()
		allUIDs, err := c.UidSearch(criteria)
		if err != nil {
			t.Fatalf("Failed to search for UIDs: %v", err)
		}
		if len(allUIDs) == 0 {
			t.Fatal("No UIDs found")
		}
		highestUID := allUIDs[len(allUIDs)-1]

		latestUID := int64(highestUID)
		err = db.SetFolderSyncInfo(ctx, pool, userID, folderName, &latestUID)
		if err != nil {
			t.Fatalf("Failed to update sync info: %v", err)
		}

		updatedSyncInfo, err := db.GetFolderSyncInfo(ctx, pool, userID, folderName)
		if err != nil {
			t.Fatalf("Failed to get updated sync info: %v", err)
		}

		result, ok := service.tryIncrementalSync(ctx, c, userID, folderName, updatedSyncInfo)
		assert.True(t, ok)
		assert.True(t, result.shouldReturn, "shouldReturn should be true when no new messages")
		assert.Empty(t, result.uidsToSync)
	})
}

func TestProcessIncrementalMessage(t *testing.T) {
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

	encryptor := testutil.GetTestEncryptor(t)
	service := NewService(pool, NewPool(), encryptor)
	defer service.Close()

	userID, err := db.GetOrCreateUser(ctx, pool, "process-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	folderName := "INBOX"

	t.Run("creates new thread for new message", func(t *testing.T) {
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
		assert.NoError(t, err)

		thread, err := db.GetThreadByStableID(ctx, pool, userID, messageID)
		assert.NoError(t, err)
		assert.Equal(t, subject, thread.Subject)

		msg, err := db.GetMessageByMessageID(ctx, pool, userID, messageID)
		assert.NoError(t, err)
		assert.Equal(t, thread.ID, msg.ThreadID)
	})

	t.Run("uses existing thread when message already exists", func(t *testing.T) {
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
		assert.NoError(t, err)

		msg, err := db.GetMessageByMessageID(ctx, pool, userID, messageID)
		assert.NoError(t, err)
		assert.Equal(t, thread.ID, msg.ThreadID)
	})

	t.Run("matches existing thread by being a reply (message exists in DB)", func(t *testing.T) {
		rootMessageID := "<root@test>"
		thread := &models.Thread{
			UserID:         userID,
			StableThreadID: rootMessageID,
			Subject:        "Root Thread",
		}
		err := db.SaveThread(ctx, pool, thread)
		if err != nil {
			t.Fatalf("Failed to save thread: %v", err)
		}

		replyMessageID := "<reply@test>"
		existingMsg := &models.Message{
			UserID:          userID,
			ThreadID:        thread.ID,
			MessageIDHeader: replyMessageID,
			IMAPFolderName:  folderName,
			IMAPUID:         4,
			Subject:         "Re: Root Thread",
			BodyText:        "Original body",
			UnsafeBodyHTML:  "<p>Original body</p>",
		}
		err = db.SaveMessage(ctx, pool, existingMsg)
		if err != nil {
			t.Fatalf("Failed to save existing message: %v", err)
		}

		imapMsg := &imap.Message{
			Uid: 4,
			Envelope: &imap.Envelope{
				MessageId: replyMessageID,
				Subject:   "Re: Root Thread",
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
		assert.NoError(t, err)

		msg, err := db.GetMessageByMessageID(ctx, pool, userID, replyMessageID)
		assert.NoError(t, err)
		assert.Equal(t, thread.ID, msg.ThreadID)
	})

	t.Run("skips message without Message-ID", func(t *testing.T) {
		imapMsg := &imap.Message{
			Uid: 3,
			Envelope: &imap.Envelope{
				Subject: "No Message-ID",
				Date:    time.Now(),
			},
			Flags: []string{imap.SeenFlag},
		}

		err := service.processIncrementalMessage(ctx, imapMsg, userID, folderName)
		assert.NoError(t, err, "should not fail for message without Message-ID")
	})
}
