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
	getClientResult    imap.IMAPClient
	getClientErr       error
	getClientCalled    bool
	getClientCallCount int
	getClientUserID    string
	getClientServer    string
	getClientUser      string
	getClientPass      string
	removeClientCalled map[string]bool
	// For retry scenarios: the first call returns one client, the second call returns another
	retryClient    imap.IMAPClient
	retryClientErr error
}

func (m *mockIMAPPool) GetClient(userID, server, username, password string) (imap.IMAPClient, error) {
	m.getClientCalled = true
	m.getClientCallCount++
	m.getClientUserID = userID
	m.getClientServer = server
	m.getClientUser = username
	m.getClientPass = password

	// If this is a retry (second call) and we have a retry client configured, use it
	if m.getClientCallCount > 1 && m.retryClient != nil {
		return m.retryClient, m.retryClientErr
	}

	return m.getClientResult, m.getClientErr
}

func (m *mockIMAPPool) RemoveClient(userID string) {
	// Track removals for testing
	if m.removeClientCalled == nil {
		m.removeClientCalled = make(map[string]bool)
	}
	m.removeClientCalled[userID] = true
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

	t.Run("recovers from broken pipe error with retry", func(t *testing.T) {
		// First client returns broken pipe error
		brokenClient := &mockIMAPClient{
			listFoldersResult: nil,
			listFoldersErr:    fmt.Errorf("failed to list folders: write tcp 192.168.1.191:51443->37.27.245.171:993: write: broken pipe"),
		}

		// Retry client succeeds
		retryClient := &mockIMAPClient{
			listFoldersResult: []string{"INBOX", "Sent"},
			listFoldersErr:    nil,
		}

		mockPool := &mockIMAPPool{
			getClientResult: brokenClient,
			getClientErr:    nil,
			retryClient:     retryClient,
			retryClientErr:  nil,
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 after retry, got %d", rr.Code)
		}

		// Verify RemoveClient was called
		if mockPool.removeClientCalled == nil || !mockPool.removeClientCalled[userID] {
			t.Error("Expected RemoveClient to be called for broken connection")
		}

		// Verify GetClient was called twice (initial + retry)
		if mockPool.getClientCallCount != 2 {
			t.Errorf("Expected GetClient to be called 2 times, got %d", mockPool.getClientCallCount)
		}

		// Verify response contains folders
		var response []models.Folder
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) != 2 {
			t.Errorf("Expected 2 folders, got %d", len(response))
		}
	})

	t.Run("handles connection reset error with retry", func(t *testing.T) {
		// First client returns connection reset error
		brokenClient := &mockIMAPClient{
			listFoldersResult: nil,
			listFoldersErr:    fmt.Errorf("failed to list folders: connection reset by peer"),
		}

		// Retry client succeeds
		retryClient := &mockIMAPClient{
			listFoldersResult: []string{"INBOX"},
			listFoldersErr:    nil,
		}

		mockPool := &mockIMAPPool{
			getClientResult: brokenClient,
			getClientErr:    nil,
			retryClient:     retryClient,
			retryClientErr:  nil,
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 after retry, got %d", rr.Code)
		}

		// Verify RemoveClient was called
		if mockPool.removeClientCalled == nil || !mockPool.removeClientCalled[userID] {
			t.Error("Expected RemoveClient to be called for broken connection")
		}
	})

	t.Run("handles EOF error with retry", func(t *testing.T) {
		// First client returns EOF error
		brokenClient := &mockIMAPClient{
			listFoldersResult: nil,
			listFoldersErr:    fmt.Errorf("failed to list folders: EOF"),
		}

		// Retry client succeeds
		retryClient := &mockIMAPClient{
			listFoldersResult: []string{"INBOX", "Drafts"},
			listFoldersErr:    nil,
		}

		mockPool := &mockIMAPPool{
			getClientResult: brokenClient,
			getClientErr:    nil,
			retryClient:     retryClient,
			retryClientErr:  nil,
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 after retry, got %d", rr.Code)
		}
	})

	t.Run("returns error if retry also fails", func(t *testing.T) {
		// First client returns broken pipe error
		brokenClient := &mockIMAPClient{
			listFoldersResult: nil,
			listFoldersErr:    fmt.Errorf("failed to list folders: write: broken pipe"),
		}

		// Retry client also fails
		retryClient := &mockIMAPClient{
			listFoldersResult: nil,
			listFoldersErr:    fmt.Errorf("failed to list folders: connection refused"),
		}

		mockPool := &mockIMAPPool{
			getClientResult: brokenClient,
			getClientErr:    nil,
			retryClient:     retryClient,
			retryClientErr:  nil,
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 when retry also fails, got %d", rr.Code)
		}

		// Verify RemoveClient was called
		if mockPool.removeClientCalled == nil || !mockPool.removeClientCalled[userID] {
			t.Error("Expected RemoveClient to be called for broken connection")
		}

		// Verify GetClient was called twice (initial + retry)
		if mockPool.getClientCallCount != 2 {
			t.Errorf("Expected GetClient to be called 2 times, got %d", mockPool.getClientCallCount)
		}
	})

	t.Run("does not retry on non-connection errors", func(t *testing.T) {
		// Client returns a non-connection error
		mockClient := &mockIMAPClient{
			listFoldersResult: nil,
			listFoldersErr:    fmt.Errorf("failed to list folders: authentication failed"),
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

		// Verify RemoveClient was NOT called for non-connection errors
		if mockPool.removeClientCalled != nil && mockPool.removeClientCalled[userID] {
			t.Error("Expected RemoveClient NOT to be called for non-connection errors")
		}

		// Verify GetClient was called only once
		if mockPool.getClientCallCount != 1 {
			t.Errorf("Expected GetClient to be called 1 time, got %d", mockPool.getClientCallCount)
		}
	})
}
