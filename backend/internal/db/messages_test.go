package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestSaveAndGetMessage(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	thread := &models.Thread{
		UserID:         userID,
		StableThreadID: "test-thread",
		Subject:        "Test Subject",
	}
	err = SaveThread(ctx, pool, thread)
	if err != nil {
		t.Fatalf("SaveThread failed: %v", err)
	}

	tests := []struct {
		name        string
		setup       func() *models.Message
		expectError bool
		checkResult func(*testing.T, *models.Message)
	}{
		{
			name: "saves and retrieves message",
			setup: func() *models.Message {
				now := time.Now()
				return &models.Message{
					ThreadID:        thread.ID,
					UserID:          userID,
					IMAPUID:         100,
					IMAPFolderName:  "INBOX",
					MessageIDHeader: "msg-id-123",
					FromAddress:     "sender@example.com",
					ToAddresses:     []string{"recipient@example.com"},
					CCAddresses:     []string{"cc@example.com"},
					Subject:         "Test Subject",
					SentAt:          &now,
					IsRead:          false,
					IsStarred:       true,
				}
			},
			expectError: false,
			checkResult: func(t *testing.T, retrieved *models.Message) {
				assert.Equal(t, "msg-id-123", retrieved.MessageIDHeader)
				assert.Equal(t, "sender@example.com", retrieved.FromAddress)
				assert.True(t, retrieved.IsStarred)
			},
		},
		{
			name: "updates existing message",
			setup: func() *models.Message {
				msg := &models.Message{
					ThreadID:        thread.ID,
					UserID:          userID,
					IMAPUID:         200,
					IMAPFolderName:  "INBOX",
					MessageIDHeader: "msg-id-456",
					Subject:         "Original Subject",
					IsRead:          false,
				}
				_ = SaveMessage(ctx, pool, msg)
				msg.Subject = "Updated Subject"
				msg.IsRead = true
				return msg
			},
			expectError: false,
			checkResult: func(t *testing.T, retrieved *models.Message) {
				assert.Equal(t, "Updated Subject", retrieved.Subject)
				assert.True(t, retrieved.IsRead)
			},
		},
		{
			name: "returns error for non-existent message",
			setup: func() *models.Message {
				return nil // Not used for this test
			},
			expectError: true,
			checkResult: func(t *testing.T, retrieved *models.Message) {
				// Error case, no need to check result
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.setup()
			if msg != nil {
				err := SaveMessage(ctx, pool, msg)
				assert.NoError(t, err)

				retrieved, err := GetMessageByUID(ctx, pool, userID, "INBOX", msg.IMAPUID)
				if tt.expectError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, retrieved)
				}
			} else {
				// Test error case
				_, err := GetMessageByUID(ctx, pool, userID, "INBOX", 99999)
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrMessageNotFound))
			}
		})
	}
}

func TestGetMessagesForThread(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	thread := &models.Thread{
		UserID:         userID,
		StableThreadID: "test-thread",
		Subject:        "Test Subject",
	}
	err = SaveThread(ctx, pool, thread)
	if err != nil {
		t.Fatalf("SaveThread failed: %v", err)
	}

	now := time.Now()
	msg1 := &models.Message{
		ThreadID:        thread.ID,
		UserID:          userID,
		IMAPUID:         1,
		IMAPFolderName:  "INBOX",
		MessageIDHeader: "msg-1",
		Subject:         "Test Subject",
		SentAt:          &now,
	}
	msg2 := &models.Message{
		ThreadID:        thread.ID,
		UserID:          userID,
		IMAPUID:         2,
		IMAPFolderName:  "INBOX",
		MessageIDHeader: "msg-2",
		Subject:         "Re: Test Subject",
		SentAt:          &now,
	}

	err = SaveMessage(ctx, pool, msg1)
	assert.NoError(t, err)
	err = SaveMessage(ctx, pool, msg2)
	assert.NoError(t, err)

	messages, err := GetMessagesForThread(ctx, pool, thread.ID)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
}

func TestSaveAndGetAttachment(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	thread := &models.Thread{
		UserID:         userID,
		StableThreadID: "test-thread",
		Subject:        "Test Subject",
	}
	err = SaveThread(ctx, pool, thread)
	if err != nil {
		t.Fatalf("SaveThread failed: %v", err)
	}

	msg := &models.Message{
		ThreadID:        thread.ID,
		UserID:          userID,
		IMAPUID:         1,
		IMAPFolderName:  "INBOX",
		MessageIDHeader: "msg-1",
		Subject:         "Test Subject",
	}
	err = SaveMessage(ctx, pool, msg)
	assert.NoError(t, err)

	attachment := &models.Attachment{
		MessageID: msg.ID,
		Filename:  "test.pdf",
		MimeType:  "application/pdf",
		SizeBytes: 1024,
		IsInline:  false,
	}

	err = SaveAttachment(ctx, pool, attachment)
	assert.NoError(t, err)
	assert.NotEmpty(t, attachment.ID)

	attachments, err := GetAttachmentsForMessage(ctx, pool, msg.ID)
	assert.NoError(t, err)
	assert.Len(t, attachments, 1)
	assert.Equal(t, "test.pdf", attachments[0].Filename)
}
