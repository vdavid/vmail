package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
)

func TestFoldersHandler_GetFolders(t *testing.T) {
	pool := setupTestPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()
	defer cleanupTestPool(t, pool)

	encryptor := getTestEncryptor(t)
	imapPool := imap.NewPool()
	defer imapPool.Close()
	handler := NewFoldersHandler(pool, encryptor, imapPool)

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/folders", nil)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("returns 404 when user settings not found", func(t *testing.T) {
		email := "newuser@example.com"

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rr.Code)
		}
	})

	// Note: Testing the actual IMAP connection would require a real IMAP server
	// or a mock. For now, we test the error handling paths.
	// Integration tests would test the full IMAP connection flow.
}

// mockIMAPClient is a mock implementation of IMAPClient for testing
type mockIMAPClient struct {
	listFoldersResult []string
	listFoldersErr    error
}

func (m *mockIMAPClient) ListFolders() ([]string, error) {
	return m.listFoldersResult, m.listFoldersErr
}

// mockIMAPPool is a mock implementation of IMAPPool for testing
type mockIMAPPool struct {
	getClientResult imap.IMAPClient
	getClientErr    error
	getClientCalled bool
	getClientUserID string
	getClientServer string
	getClientUser   string
	getClientPass   string
}

func (m *mockIMAPPool) GetClient(userID, server, username, password string) (imap.IMAPClient, error) {
	m.getClientCalled = true
	m.getClientUserID = userID
	m.getClientServer = server
	m.getClientUser = username
	m.getClientPass = password
	return m.getClientResult, m.getClientErr
}

func (m *mockIMAPPool) Close() {}

func TestFoldersHandler_WithMocks(t *testing.T) {
	pool := setupTestPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()
	defer cleanupTestPool(t, pool)

	encryptor := getTestEncryptor(t)
	email := "folders-test@example.com"

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
		ArchiveFolderName:        "Archive",
		SentFolderName:           "Sent",
		DraftsFolderName:         "Drafts",
		TrashFolderName:          "Trash",
		SpamFolderName:           "Spam",
	}
	if err := db.SaveUserSettings(ctx, pool, settings); err != nil {
		t.Fatalf("Failed to save settings: %v", err)
	}

	t.Run("returns folders from IMAP", func(t *testing.T) {
		mockClient := &mockIMAPClient{
			listFoldersResult: []string{"INBOX", "Sent", "Drafts", "Archive"},
			listFoldersErr:    nil,
		}

		mockPool := &mockIMAPPool{
			getClientResult: mockClient,
			getClientErr:    nil,
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Verify that we called GetClient with the correct parameters
		if !mockPool.getClientCalled {
			t.Error("Expected GetClient to be called")
		}
		if mockPool.getClientUserID != userID {
			t.Errorf("Expected userID %s, got %s", userID, mockPool.getClientUserID)
		}
		if mockPool.getClientServer != "imap.test.com" {
			t.Errorf("Expected server 'imap.test.com', got %s", mockPool.getClientServer)
		}

		// Verify response contains folders (response is an array, not an object)
		var response []models.Folder
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 4 {
			t.Errorf("Expected 4 folders, got %d", len(response))
		}

		expectedFolders := []string{"INBOX", "Sent", "Drafts", "Archive"}
		for i, expected := range expectedFolders {
			if response[i].Name != expected {
				t.Errorf("Expected folder %d to be '%s', got '%s'", i, expected, response[i].Name)
			}
		}
	})

	t.Run("handles IMAP connection error", func(t *testing.T) {
		mockPool := &mockIMAPPool{
			getClientResult: nil,
			getClientErr:    fmt.Errorf("connection failed"),
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})

	t.Run("handles ListFolders error", func(t *testing.T) {
		mockClient := &mockIMAPClient{
			listFoldersResult: nil,
			listFoldersErr:    fmt.Errorf("list folders failed"),
		}

		mockPool := &mockIMAPPool{
			getClientResult: mockClient,
			getClientErr:    nil,
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})
}
