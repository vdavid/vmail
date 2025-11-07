package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/models"
)

func TestSaveAndGetThread(t *testing.T) {
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
