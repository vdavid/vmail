package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestSettingsHandler_GetSettings(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	handler := NewSettingsHandler(pool, encryptor)

	t.Run("returns 404 for user without settings", func(t *testing.T) {
		email := "new-user@example.com"

		req := httptest.NewRequest("GET", "/api/v1/settings", nil)
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.GetSettings(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}
	})

	t.Run("returns settings for user with settings", func(t *testing.T) {
		email := "setupuser@example.com"

		ctx := context.Background()
		userID, err := db.GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		encryptedIMAPPassword, _ := encryptor.Encrypt("imap_pass_123")
		encryptedSMTPPassword, _ := encryptor.Encrypt("smtp_pass_456")

		settings := &models.UserSettings{
			UserID:                   userID,
			UndoSendDelaySeconds:     30,
			PaginationThreadsPerPage: 50,
			IMAPServerHostname:       "imap.test.com",
			IMAPUsername:             "test_user",
			EncryptedIMAPPassword:    encryptedIMAPPassword,
			SMTPServerHostname:       "smtp.test.com",
			SMTPUsername:             "test_user",
			EncryptedSMTPPassword:    encryptedSMTPPassword,
		}
		if err := db.SaveUserSettings(ctx, pool, settings); err != nil {
			t.Fatalf("Failed to save settings: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/v1/settings", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetSettings(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response models.UserSettingsResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.IMAPServerHostname != "imap.test.com" {
			t.Errorf("Expected IMAPServerHostname 'imap.test.com', got %s", response.IMAPServerHostname)
		}
		if response.UndoSendDelaySeconds != 30 {
			t.Errorf("Expected UndoSendDelaySeconds 30, got %d", response.UndoSendDelaySeconds)
		}
		if !response.IMAPPasswordSet {
			t.Error("Expected IMAPPasswordSet to be true")
		}
		if !response.SMTPPasswordSet {
			t.Error("Expected SMTPPasswordSet to be true")
		}
	})

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/settings", nil)

		rr := httptest.NewRecorder()
		handler.GetSettings(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})
}

func TestSettingsHandler_PostSettings(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	handler := NewSettingsHandler(pool, encryptor)

	t.Run("saves new settings successfully", func(t *testing.T) {
		email := "new-user@example.com"

		reqBody := models.UserSettingsRequest{
			UndoSendDelaySeconds:     25,
			PaginationThreadsPerPage: 75,
			IMAPServerHostname:       "imap.new.com",
			IMAPUsername:             "new-user",
			IMAPPassword:             "imap_password_123",
			SMTPServerHostname:       "smtp.new.com",
			SMTPUsername:             "new-user",
			SMTPPassword:             "smtp_password_456",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		userID, _ := db.GetOrCreateUser(context.Background(), pool, email)
		savedSettings, err := db.GetUserSettings(context.Background(), pool, userID)
		if err != nil {
			t.Fatalf("Failed to get saved settings: %v", err)
		}

		if savedSettings.IMAPServerHostname != "imap.new.com" {
			t.Errorf("Expected IMAPServerHostname 'imap.new.com', got %s", savedSettings.IMAPServerHostname)
		}

		decryptedIMAPPassword, _ := encryptor.Decrypt(savedSettings.EncryptedIMAPPassword)
		if decryptedIMAPPassword != "imap_password_123" {
			t.Error("IMAP password was not encrypted/decrypted correctly")
		}

		decryptedSMTPPassword, _ := encryptor.Decrypt(savedSettings.EncryptedSMTPPassword)
		if decryptedSMTPPassword != "smtp_password_456" {
			t.Error("SMTP password was not encrypted/decrypted correctly")
		}
	})

	t.Run("updates existing settings", func(t *testing.T) {
		email := "updateuser@example.com"

		ctx := context.Background()
		userID, _ := db.GetOrCreateUser(ctx, pool, email)

		initialSettings := &models.UserSettings{
			UserID:                   userID,
			UndoSendDelaySeconds:     20,
			PaginationThreadsPerPage: 100,
			IMAPServerHostname:       "old.imap.com",
			IMAPUsername:             "old_user",
			EncryptedIMAPPassword:    []byte("old_encrypted"),
			SMTPServerHostname:       "old.smtp.com",
			SMTPUsername:             "old_user",
			EncryptedSMTPPassword:    []byte("old_encrypted"),
		}
		err := db.SaveUserSettings(ctx, pool, initialSettings)
		if err != nil {
			t.Fatalf("Failed to save initial settings: %v", err)
		}

		reqBody := models.UserSettingsRequest{
			UndoSendDelaySeconds:     40,
			PaginationThreadsPerPage: 200,
			IMAPServerHostname:       "new.imap.com",
			IMAPUsername:             "new_user",
			IMAPPassword:             "new_imap_password",
			SMTPServerHostname:       "new.smtp.com",
			SMTPUsername:             "new_user",
			SMTPPassword:             "new_smtp_password",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		updatedSettings, _ := db.GetUserSettings(context.Background(), pool, userID)
		if updatedSettings.IMAPServerHostname != "new.imap.com" {
			t.Error("Settings were not updated")
		}
	})

	t.Run("returns 400 for invalid request body", func(t *testing.T) {
		email := "user@example.com"

		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader([]byte("invalid json")))
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/settings", nil)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("updates settings without passwords when passwords are empty", func(t *testing.T) {
		email := "updatewithoutpass@example.com"

		ctx := context.Background()
		userID, _ := db.GetOrCreateUser(ctx, pool, email)

		encryptedIMAPPassword, _ := encryptor.Encrypt("original_imap_pass")
		encryptedSMTPPassword, _ := encryptor.Encrypt("original_smtp_pass")

		initialSettings := &models.UserSettings{
			UserID:                   userID,
			UndoSendDelaySeconds:     20,
			PaginationThreadsPerPage: 100,
			IMAPServerHostname:       "old.imap.com",
			IMAPUsername:             "old_user",
			EncryptedIMAPPassword:    encryptedIMAPPassword,
			SMTPServerHostname:       "old.smtp.com",
			SMTPUsername:             "old_user",
			EncryptedSMTPPassword:    encryptedSMTPPassword,
		}
		err := db.SaveUserSettings(ctx, pool, initialSettings)
		if err != nil {
			t.Fatalf("Failed to save initial settings: %v", err)
		}

		// Update settings without providing passwords
		reqBody := models.UserSettingsRequest{
			UndoSendDelaySeconds:     40,
			PaginationThreadsPerPage: 200,
			IMAPServerHostname:       "new.imap.com",
			IMAPUsername:             "new_user",
			IMAPPassword:             "", // Empty password
			SMTPServerHostname:       "new.smtp.com",
			SMTPUsername:             "new_user",
			SMTPPassword:             "", // Empty password
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		updatedSettings, _ := db.GetUserSettings(context.Background(), pool, userID)
		if updatedSettings.IMAPServerHostname != "new.imap.com" {
			t.Error("Settings were not updated")
		}

		// Verify passwords were preserved
		decryptedIMAPPassword, _ := encryptor.Decrypt(updatedSettings.EncryptedIMAPPassword)
		if decryptedIMAPPassword != "original_imap_pass" {
			t.Error("IMAP password should have been preserved but was changed")
		}

		decryptedSMTPPassword, _ := encryptor.Decrypt(updatedSettings.EncryptedSMTPPassword)
		if decryptedSMTPPassword != "original_smtp_pass" {
			t.Error("SMTP password should have been preserved but was changed")
		}
	})

	t.Run("returns 400 when passwords are empty for new user", func(t *testing.T) {
		email := "newuser@example.com"

		reqBody := models.UserSettingsRequest{
			UndoSendDelaySeconds:     25,
			PaginationThreadsPerPage: 75,
			IMAPServerHostname:       "imap.new.com",
			IMAPUsername:             "new-user",
			IMAPPassword:             "", // Empty password for new user
			SMTPServerHostname:       "smtp.new.com",
			SMTPUsername:             "new-user",
			SMTPPassword:             "", // Empty password for new user
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty passwords on new user, got %d", rr.Code)
		}
	})

	t.Run("returns 500 when GetUserSettings returns non-NotFound error in PostSettings", func(t *testing.T) {
		email := "dberror-post@example.com"

		// Use a cancelled context to simulate database connection failure
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		reqBody := models.UserSettingsRequest{
			UndoSendDelaySeconds:     25,
			PaginationThreadsPerPage: 75,
			IMAPServerHostname:       "imap.new.com",
			IMAPUsername:             "new-user",
			IMAPPassword:             "imap_password_123",
			SMTPServerHostname:       "smtp.new.com",
			SMTPUsername:             "new-user",
			SMTPPassword:             "smtp_password_456",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
		reqCtx := context.WithValue(cancelledCtx, auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})

	// Note: Testing SaveUserSettings failure is difficult without mocking the database layer.
	// The error handling code path is covered by the handler implementation, but simulating
	// a database save failure in a real test environment is complex. The error handling
	// is straightforward (returns 500 on error), so we rely on integration tests and
	// the code coverage to ensure this path works correctly.

	t.Run("returns 500 when GetUserSettings returns non-NotFound error in GetSettings", func(t *testing.T) {
		email := "dberror-get@example.com"

		// Use a cancelled context to simulate database connection failure
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		req := httptest.NewRequest("GET", "/api/v1/settings", nil)
		reqCtx := context.WithValue(cancelledCtx, auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetSettings(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})

	t.Run("validates missing IMAP server hostname", func(t *testing.T) {
		email := "validation-test@example.com"

		reqBody := models.UserSettingsRequest{
			UndoSendDelaySeconds:     25,
			PaginationThreadsPerPage: 75,
			IMAPServerHostname:       "", // Missing
			IMAPUsername:             "user",
			IMAPPassword:             "password",
			SMTPServerHostname:       "smtp.test.com",
			SMTPUsername:             "user",
			SMTPPassword:             "password",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}

		bodyStr := rr.Body.String()
		if !strings.Contains(bodyStr, "IMAP server hostname is required") {
			t.Errorf("Expected error message about IMAP server hostname, got: %s", bodyStr)
		}
	})

	t.Run("validates missing IMAP username", func(t *testing.T) {
		email := "validation-test2@example.com"

		reqBody := models.UserSettingsRequest{
			UndoSendDelaySeconds:     25,
			PaginationThreadsPerPage: 75,
			IMAPServerHostname:       "imap.test.com",
			IMAPUsername:             "", // Missing
			IMAPPassword:             "password",
			SMTPServerHostname:       "smtp.test.com",
			SMTPUsername:             "user",
			SMTPPassword:             "password",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}

		bodyStr := rr.Body.String()
		if !strings.Contains(bodyStr, "IMAP username is required") {
			t.Errorf("Expected error message about IMAP username, got: %s", bodyStr)
		}
	})

	t.Run("validates missing SMTP server hostname", func(t *testing.T) {
		email := "validation-test3@example.com"

		reqBody := models.UserSettingsRequest{
			UndoSendDelaySeconds:     25,
			PaginationThreadsPerPage: 75,
			IMAPServerHostname:       "imap.test.com",
			IMAPUsername:             "user",
			IMAPPassword:             "password",
			SMTPServerHostname:       "", // Missing
			SMTPUsername:             "user",
			SMTPPassword:             "password",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}

		bodyStr := rr.Body.String()
		if !strings.Contains(bodyStr, "SMTP server hostname is required") {
			t.Errorf("Expected error message about SMTP server hostname, got: %s", bodyStr)
		}
	})

	t.Run("validates missing SMTP username", func(t *testing.T) {
		email := "validation-test4@example.com"

		reqBody := models.UserSettingsRequest{
			UndoSendDelaySeconds:     25,
			PaginationThreadsPerPage: 75,
			IMAPServerHostname:       "imap.test.com",
			IMAPUsername:             "user",
			IMAPPassword:             "password",
			SMTPServerHostname:       "smtp.test.com",
			SMTPUsername:             "", // Missing
			SMTPPassword:             "password",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.PostSettings(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}

		bodyStr := rr.Body.String()
		if !strings.Contains(bodyStr, "SMTP username is required") {
			t.Errorf("Expected error message about SMTP username, got: %s", bodyStr)
		}
	})
}

// failingResponseWriter is a ResponseWriter that fails on Write to test error handling.
type failingResponseWriterSettings struct {
	http.ResponseWriter
	writeShouldFail bool
}

func (f *failingResponseWriterSettings) Write(p []byte) (int, error) {
	if f.writeShouldFail {
		return 0, fmt.Errorf("write failed")
	}
	return f.ResponseWriter.Write(p)
}

func TestSettingsHandler_WriteResponseErrors(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	handler := NewSettingsHandler(pool, encryptor)

	t.Run("handles write failure gracefully in GetSettings", func(t *testing.T) {
		email := "write-error-get@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		req := httptest.NewRequest("GET", "/api/v1/settings", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		// Create a ResponseWriter that fails on Write
		rr := httptest.NewRecorder()
		failingWriter := &failingResponseWriterSettings{
			ResponseWriter:  rr,
			writeShouldFail: true,
		}

		handler.GetSettings(failingWriter, req)

		// The handler should handle the write error gracefully (it logs but doesn't crash)
		// We can't easily test the error path without checking logs, but we verify it doesn't panic
	})
}
