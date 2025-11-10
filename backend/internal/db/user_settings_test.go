package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestGetOrCreateUser(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	t.Run("creates new user", func(t *testing.T) {
		email := "test@example.com"

		userID, err := GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("GetOrCreateUser failed: %v", err)
		}

		if userID == "" {
			t.Fatal("Expected non-empty user ID")
		}
	})

	t.Run("returns existing user", func(t *testing.T) {
		email := "existing@example.com"

		userID1, err := GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("First GetOrCreateUser failed: %v", err)
		}

		userID2, err := GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("Second GetOrCreateUser failed: %v", err)
		}

		if userID1 != userID2 {
			t.Errorf("Expected same user ID, got %s and %s", userID1, userID2)
		}
	})
}

func TestUserSettingsExist(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	t.Run("returns false when settings don't exist", func(t *testing.T) {
		exists, err := UserSettingsExist(ctx, pool, userID)
		if err != nil {
			t.Fatalf("UserSettingsExist failed: %v", err)
		}

		if exists {
			t.Error("Expected settings to not exist")
		}
	})

	t.Run("returns true when settings exist", func(t *testing.T) {
		settings := &models.UserSettings{
			UserID:                   userID,
			UndoSendDelaySeconds:     20,
			PaginationThreadsPerPage: 100,
			IMAPServerHostname:       "imap.example.com",
			IMAPUsername:             "user@example.com",
			EncryptedIMAPPassword:    []byte("encrypted"),
			SMTPServerHostname:       "smtp.example.com",
			SMTPUsername:             "user@example.com",
			EncryptedSMTPPassword:    []byte("encrypted"),
			ArchiveFolderName:        "Archive",
			SentFolderName:           "Sent",
			DraftsFolderName:         "Drafts",
			TrashFolderName:          "Trash",
			SpamFolderName:           "Spam",
		}

		err := SaveUserSettings(ctx, pool, settings)
		if err != nil {
			t.Fatalf("SaveUserSettings failed: %v", err)
		}

		exists, err := UserSettingsExist(ctx, pool, userID)
		if err != nil {
			t.Fatalf("UserSettingsExist failed: %v", err)
		}

		if !exists {
			t.Error("Expected settings to exist")
		}
	})
}

func TestSaveAndGetUserSettings(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	t.Run("saves and retrieves settings", func(t *testing.T) {
		settings := &models.UserSettings{
			UserID:                   userID,
			UndoSendDelaySeconds:     30,
			PaginationThreadsPerPage: 50,
			IMAPServerHostname:       "imap.test.com",
			IMAPUsername:             "test_user",
			EncryptedIMAPPassword:    []byte("encrypted_imap_pass"),
			SMTPServerHostname:       "smtp.test.com",
			SMTPUsername:             "test_user",
			EncryptedSMTPPassword:    []byte("encrypted_smtp_pass"),
			ArchiveFolderName:        "MyArchive",
			SentFolderName:           "MySent",
			DraftsFolderName:         "MyDrafts",
			TrashFolderName:          "MyTrash",
			SpamFolderName:           "MySpam",
		}

		err := SaveUserSettings(ctx, pool, settings)
		if err != nil {
			t.Fatalf("SaveUserSettings failed: %v", err)
		}

		retrieved, err := GetUserSettings(ctx, pool, userID)
		if err != nil {
			t.Fatalf("GetUserSettings failed: %v", err)
		}

		if retrieved.UserID != settings.UserID {
			t.Errorf("Expected UserID %s, got %s", settings.UserID, retrieved.UserID)
		}
		if retrieved.UndoSendDelaySeconds != settings.UndoSendDelaySeconds {
			t.Errorf("Expected UndoSendDelaySeconds %d, got %d", settings.UndoSendDelaySeconds, retrieved.UndoSendDelaySeconds)
		}
		if retrieved.IMAPServerHostname != settings.IMAPServerHostname {
			t.Errorf("Expected IMAPServerHostname %s, got %s", settings.IMAPServerHostname, retrieved.IMAPServerHostname)
		}
		if string(retrieved.EncryptedIMAPPassword) != string(settings.EncryptedIMAPPassword) {
			t.Errorf("Expected EncryptedIMAPPassword %s, got %s", settings.EncryptedIMAPPassword, retrieved.EncryptedIMAPPassword)
		}
	})

	t.Run("updates existing settings", func(t *testing.T) {
		updatedSettings := &models.UserSettings{
			UserID:                   userID,
			UndoSendDelaySeconds:     60,
			PaginationThreadsPerPage: 200,
			IMAPServerHostname:       "imap.updated.com",
			IMAPUsername:             "updated_user",
			EncryptedIMAPPassword:    []byte("new_encrypted_imap"),
			SMTPServerHostname:       "smtp.updated.com",
			SMTPUsername:             "updated_user",
			EncryptedSMTPPassword:    []byte("new_encrypted_smtp"),
			ArchiveFolderName:        "NewArchive",
			SentFolderName:           "NewSent",
			DraftsFolderName:         "NewDrafts",
			TrashFolderName:          "NewTrash",
			SpamFolderName:           "NewSpam",
		}

		err := SaveUserSettings(ctx, pool, updatedSettings)
		if err != nil {
			t.Fatalf("SaveUserSettings (update) failed: %v", err)
		}

		retrieved, err := GetUserSettings(ctx, pool, userID)
		if err != nil {
			t.Fatalf("GetUserSettings failed: %v", err)
		}

		if retrieved.UndoSendDelaySeconds != 60 {
			t.Errorf("Expected updated UndoSendDelaySeconds 60, got %d", retrieved.UndoSendDelaySeconds)
		}
		if retrieved.IMAPServerHostname != "imap.updated.com" {
			t.Errorf("Expected updated IMAPServerHostname, got %s", retrieved.IMAPServerHostname)
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		_, err := GetUserSettings(ctx, pool, "00000000-0000-0000-0000-000000000000")
		if !errors.Is(err, ErrUserSettingsNotFound) {
			t.Errorf("Expected ErrUserSettingsNotFound, got %v", err)
		}
	})
}

func TestSaveUserSettingsUpdatesTimestamp(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	settings := &models.UserSettings{
		UserID:                   userID,
		UndoSendDelaySeconds:     20,
		PaginationThreadsPerPage: 100,
		IMAPServerHostname:       "imap.example.com",
		IMAPUsername:             "user",
		EncryptedIMAPPassword:    []byte("pass"),
		SMTPServerHostname:       "smtp.example.com",
		SMTPUsername:             "user",
		EncryptedSMTPPassword:    []byte("pass"),
		ArchiveFolderName:        "Archive",
		SentFolderName:           "Sent",
		DraftsFolderName:         "Drafts",
		TrashFolderName:          "Trash",
		SpamFolderName:           "Spam",
	}

	err = SaveUserSettings(ctx, pool, settings)
	if err != nil {
		t.Fatalf("SaveUserSettings failed: %v", err)
	}

	retrieved1, err := GetUserSettings(ctx, pool, userID)
	if err != nil {
		t.Fatalf("GetUserSettings failed: %v", err)
	}
	if retrieved1 == nil {
		t.Fatalf("GetUserSettings returned nil")
	}

	time.Sleep(100 * time.Millisecond)

	settings.UndoSendDelaySeconds = 30
	err = SaveUserSettings(ctx, pool, settings)
	if err != nil {
		t.Fatalf("SaveUserSettings (update) failed: %v", err)
	}

	retrieved2, err := GetUserSettings(ctx, pool, userID)
	if err != nil {
		t.Fatalf("GetUserSettings (second) failed: %v", err)
	}
	if retrieved2 == nil {
		t.Fatalf("GetUserSettings (second) returned nil")
	}

	if !retrieved2.UpdatedAt.After(retrieved1.UpdatedAt) {
		t.Error("Expected updated_at to be updated after second save")
	}
}
