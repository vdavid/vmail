package db

import (
	"context"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/models"
)

func TestSaveAndGetMessage(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()
	defer cleanupTestDB(t, pool)

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

	t.Run("saves and retrieves message", func(t *testing.T) {
		now := time.Now()
		msg := &models.Message{
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

		err := SaveMessage(ctx, pool, msg)
		if err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}

		retrieved, err := GetMessageByUID(ctx, pool, userID, "INBOX", 100)
		if err != nil {
			t.Fatalf("GetMessageByUID failed: %v", err)
		}

		if retrieved.MessageIDHeader != msg.MessageIDHeader {
			t.Errorf("Expected MessageIDHeader %s, got %s", msg.MessageIDHeader, retrieved.MessageIDHeader)
		}
		if retrieved.FromAddress != msg.FromAddress {
			t.Errorf("Expected FromAddress %s, got %s", msg.FromAddress, retrieved.FromAddress)
		}
		if !retrieved.IsStarred {
			t.Error("Expected message to be starred")
		}
	})

	t.Run("updates existing message", func(t *testing.T) {
		msg := &models.Message{
			ThreadID:        thread.ID,
			UserID:          userID,
			IMAPUID:         200,
			IMAPFolderName:  "INBOX",
			MessageIDHeader: "msg-id-456",
			Subject:         "Original Subject",
			IsRead:          false,
		}

		err := SaveMessage(ctx, pool, msg)
		if err != nil {
			t.Fatalf("SaveMessage failed: %v", err)
		}

		msg.Subject = "Updated Subject"
		msg.IsRead = true
		err = SaveMessage(ctx, pool, msg)
		if err != nil {
			t.Fatalf("SaveMessage (update) failed: %v", err)
		}

		retrieved, err := GetMessageByUID(ctx, pool, userID, "INBOX", 200)
		if err != nil {
			t.Fatalf("GetMessageByUID failed: %v", err)
		}

		if retrieved.Subject != "Updated Subject" {
			t.Errorf("Expected updated Subject, got %s", retrieved.Subject)
		}
		if !retrieved.IsRead {
			t.Error("Expected message to be read")
		}
	})

	t.Run("returns error for non-existent message", func(t *testing.T) {
		_, err := GetMessageByUID(ctx, pool, userID, "INBOX", 99999)
		if err == nil {
			t.Error("Expected error for non-existent message")
		}
		if err != ErrMessageNotFound {
			t.Errorf("Expected ErrMessageNotFound, got %v", err)
		}
	})
}

func TestGetMessagesForThread(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()
	defer cleanupTestDB(t, pool)

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
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}
	err = SaveMessage(ctx, pool, msg2)
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	t.Run("returns all messages for thread", func(t *testing.T) {
		messages, err := GetMessagesForThread(ctx, pool, thread.ID)
		if err != nil {
			t.Fatalf("GetMessagesForThread failed: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}
	})
}

func TestSaveAndGetAttachment(t *testing.T) {
	pool := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()
	defer cleanupTestDB(t, pool)

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
	if err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	t.Run("saves and retrieves attachment", func(t *testing.T) {
		attachment := &models.Attachment{
			MessageID: msg.ID,
			Filename:  "test.pdf",
			MimeType:  "application/pdf",
			SizeBytes: 1024,
			IsInline:  false,
		}

		err := SaveAttachment(ctx, pool, attachment)
		if err != nil {
			t.Fatalf("SaveAttachment failed: %v", err)
		}

		if attachment.ID == "" {
			t.Error("Expected attachment ID to be set")
		}

		attachments, err := GetAttachmentsForMessage(ctx, pool, msg.ID)
		if err != nil {
			t.Fatalf("GetAttachmentsForMessage failed: %v", err)
		}

		if len(attachments) != 1 {
			t.Errorf("Expected 1 attachment, got %d", len(attachments))
		}

		if attachments[0].Filename != "test.pdf" {
			t.Errorf("Expected filename test.pdf, got %s", attachments[0].Filename)
		}
	})
}
