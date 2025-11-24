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

func TestUserSettingsExist(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	email := "test@example.com"
	userID, err := GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("GetOrCreateUser failed: %v", err)
	}

	tests := []struct {
		name     string
		setup    func()
		expected bool
	}{
		{
			name:     "returns false when settings don't exist",
			setup:    func() {}, // No setup needed
			expected: false,
		},
		{
			name: "returns true when settings exist",
			setup: func() {
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
				}
				_ = SaveUserSettings(ctx, pool, settings)
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			exists, err := UserSettingsExist(ctx, pool, userID)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, exists)
		})
	}
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

	tests := []struct {
		name        string
		setup       func() *models.UserSettings
		expectError bool
		checkResult func(*testing.T, *models.UserSettings)
	}{
		{
			name: "saves and retrieves settings",
			setup: func() *models.UserSettings {
				return &models.UserSettings{
					UserID:                   userID,
					UndoSendDelaySeconds:     30,
					PaginationThreadsPerPage: 50,
					IMAPServerHostname:       "imap.test.com",
					IMAPUsername:             "test_user",
					EncryptedIMAPPassword:    []byte("encrypted_imap_pass"),
					SMTPServerHostname:       "smtp.test.com",
					SMTPUsername:             "test_user",
					EncryptedSMTPPassword:    []byte("encrypted_smtp_pass"),
				}
			},
			expectError: false,
			checkResult: func(t *testing.T, retrieved *models.UserSettings) {
				assert.Equal(t, userID, retrieved.UserID)
				assert.Equal(t, 30, retrieved.UndoSendDelaySeconds)
				assert.Equal(t, "imap.test.com", retrieved.IMAPServerHostname)
				assert.Equal(t, []byte("encrypted_imap_pass"), retrieved.EncryptedIMAPPassword)
			},
		},
		{
			name: "updates existing settings",
			setup: func() *models.UserSettings {
				// First save initial settings
				initial := &models.UserSettings{
					UserID:                   userID,
					UndoSendDelaySeconds:     30,
					PaginationThreadsPerPage: 50,
					IMAPServerHostname:       "imap.test.com",
					IMAPUsername:             "test_user",
					EncryptedIMAPPassword:    []byte("encrypted_imap_pass"),
					SMTPServerHostname:       "smtp.test.com",
					SMTPUsername:             "test_user",
					EncryptedSMTPPassword:    []byte("encrypted_smtp_pass"),
				}
				_ = SaveUserSettings(ctx, pool, initial)

				// Return updated settings
				return &models.UserSettings{
					UserID:                   userID,
					UndoSendDelaySeconds:     60,
					PaginationThreadsPerPage: 200,
					IMAPServerHostname:       "imap.updated.com",
					IMAPUsername:             "updated_user",
					EncryptedIMAPPassword:    []byte("new_encrypted_imap"),
					SMTPServerHostname:       "smtp.updated.com",
					SMTPUsername:             "updated_user",
					EncryptedSMTPPassword:    []byte("new_encrypted_smtp"),
				}
			},
			expectError: false,
			checkResult: func(t *testing.T, retrieved *models.UserSettings) {
				assert.Equal(t, 60, retrieved.UndoSendDelaySeconds)
				assert.Equal(t, "imap.updated.com", retrieved.IMAPServerHostname)
			},
		},
		{
			name: "returns error for non-existent user",
			setup: func() *models.UserSettings {
				return nil // Not used for this test
			},
			expectError: true,
			checkResult: func(t *testing.T, retrieved *models.UserSettings) {
				// Error case, no need to check result
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := tt.setup()
			if settings != nil {
				err := SaveUserSettings(ctx, pool, settings)
				assert.NoError(t, err)

				retrieved, err := GetUserSettings(ctx, pool, userID)
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
				_, err := GetUserSettings(ctx, pool, "00000000-0000-0000-0000-000000000000")
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrUserSettingsNotFound))
			}
		})
	}
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
	}

	err = SaveUserSettings(ctx, pool, settings)
	assert.NoError(t, err)

	retrieved1, err := GetUserSettings(ctx, pool, userID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved1)

	time.Sleep(100 * time.Millisecond)

	settings.UndoSendDelaySeconds = 30
	err = SaveUserSettings(ctx, pool, settings)
	assert.NoError(t, err)

	retrieved2, err := GetUserSettings(ctx, pool, userID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved2)

	assert.True(t, retrieved2.UpdatedAt.After(retrieved1.UpdatedAt), "updated_at should be updated after second save")
}
