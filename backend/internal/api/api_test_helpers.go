package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
)

// getTestEncryptor creates a test encryptor with a deterministic key for testing.
func getTestEncryptor(t *testing.T) *crypto.Encryptor {
	t.Helper()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, err := crypto.NewEncryptor(base64Key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	return encryptor
}

// setupTestUserAndSettings creates a test user and saves their settings.
// Returns the userID for use in tests.
func setupTestUserAndSettings(t *testing.T, pool *pgxpool.Pool, encryptor *crypto.Encryptor, email string) string {
	t.Helper()
	ctx := context.Background()
	userID, err := db.GetOrCreateUser(ctx, pool, email)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	encryptedIMAPPassword, _ := encryptor.Encrypt("imap_pass")
	encryptedSMTPPassword, _ := encryptor.Encrypt("smtp_pass")

	settings := &models.UserSettings{
		UserID:                   userID,
		UndoSendDelaySeconds:     20,
		PaginationThreadsPerPage: 100,
		IMAPServerHostname:       "imap.test.com",
		IMAPUsername:             "user",
		EncryptedIMAPPassword:    encryptedIMAPPassword,
		SMTPServerHostname:       "smtp.test.com",
		SMTPUsername:             "user",
		EncryptedSMTPPassword:    encryptedSMTPPassword,
	}
	if err := db.SaveUserSettings(ctx, pool, settings); err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}
	return userID
}

// createRequestWithUser creates an HTTP request with user email in context.
func createRequestWithUser(method, url, email string) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
	return req.WithContext(ctx)
}
