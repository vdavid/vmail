package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestFoldersHandler_GetFolders(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

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

	t.Run("returns 500 when GetOrCreateUser returns an error", func(t *testing.T) {
		email := "dberror@example.com"

		// Use a cancelled context to simulate database connection failure
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()
		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(cancelledCtx, auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})
}

// mockIMAPClient is a mock implementation of IMAPClient for testing
type mockIMAPClient struct {
	listFoldersResult []*models.Folder
	listFoldersErr    error
}

func (m *mockIMAPClient) ListFolders() ([]*models.Folder, error) {
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

// callGetFolders is a helper function that sets up and calls GetFolders handler.
// It returns the response recorder for assertions.
func callGetFolders(t *testing.T, handler *FoldersHandler, email string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/v1/folders", nil)
	reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
	req = req.WithContext(reqCtx)

	rr := httptest.NewRecorder()
	handler.GetFolders(rr, req)
	return rr
}

// testRetryScenario is a helper function for testing retry scenarios with broken connections.
// It sets up a broken client, a retry client, calls GetFolders, and verifies RemoveClient was called.
// Returns the response recorder and mock pool for additional assertions.
func testRetryScenario(t *testing.T, pool *pgxpool.Pool, encryptor *crypto.Encryptor, email, userID string, brokenClientErr error, retryClient *mockIMAPClient) (*httptest.ResponseRecorder, *mockIMAPPool) {
	t.Helper()
	brokenClient := &mockIMAPClient{
		listFoldersResult: nil,
		listFoldersErr:    brokenClientErr,
	}

	mockPool := &mockIMAPPool{
		getClientResult: brokenClient,
		getClientErr:    nil,
		retryClient:     retryClient,
		retryClientErr:  nil,
	}

	handler := NewFoldersHandler(pool, encryptor, mockPool)
	rr := callGetFolders(t, handler, email)

	// Verify RemoveClient was called
	if mockPool.removeClientCalled == nil || !mockPool.removeClientCalled[userID] {
		t.Error("Expected RemoveClient to be called for broken connection")
	}

	return rr, mockPool
}

func TestFoldersHandler_WithMocks(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	email := "folders-test@example.com"
	userID := setupTestUserAndSettings(t, pool, encryptor, email)

	t.Run("returns folders from IMAP", func(t *testing.T) {
		mockClient := &mockIMAPClient{
			listFoldersResult: []*models.Folder{
				{Name: "INBOX", Role: "inbox"},
				{Name: "Sent", Role: "sent"},
				{Name: "Drafts", Role: "drafts"},
				{Name: "Archive", Role: "archive"},
			},
			listFoldersErr: nil,
		}

		mockPool := &mockIMAPPool{
			getClientResult: mockClient,
			getClientErr:    nil,
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)
		rr := callGetFolders(t, handler, email)

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

		expectedFolders := []struct {
			name string
			role string
		}{
			{"INBOX", "inbox"},
			{"Sent", "sent"},
			{"Drafts", "drafts"},
			{"Archive", "archive"},
		}
		for i, expected := range expectedFolders {
			if response[i].Name != expected.name {
				t.Errorf("Expected folder %d name to be '%s', got '%s'", i, expected.name, response[i].Name)
			}
			if response[i].Role != expected.role {
				t.Errorf("Expected folder %d role to be '%s', got '%s'", i, expected.role, response[i].Role)
			}
		}
	})

	t.Run("handles IMAP connection error", func(t *testing.T) {
		mockPool := &mockIMAPPool{
			getClientResult: nil,
			getClientErr:    fmt.Errorf("connection failed"),
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)
		rr := callGetFolders(t, handler, email)

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
		rr := callGetFolders(t, handler, email)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})

	t.Run("recovers from broken pipe error with retry", func(t *testing.T) {
		retryClient := &mockIMAPClient{
			listFoldersResult: []*models.Folder{
				{Name: "INBOX", Role: "inbox"},
				{Name: "Sent", Role: "sent"},
			},
			listFoldersErr: nil,
		}

		rr, mockPool := testRetryScenario(t, pool, encryptor, email, userID,
			fmt.Errorf("failed to list folders: write tcp 192.168.1.191:51443->37.27.245.171:993: write: broken pipe"),
			retryClient)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 after retry, got %d", rr.Code)
		}

		// Verify GetClient was called twice (initial plus retry)
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
		retryClient := &mockIMAPClient{
			listFoldersResult: []*models.Folder{
				{Name: "INBOX", Role: "inbox"},
			},
			listFoldersErr: nil,
		}

		rr, _ := testRetryScenario(t, pool, encryptor, email, userID,
			fmt.Errorf("failed to list folders: connection reset by peer"),
			retryClient)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 after retry, got %d", rr.Code)
		}
	})

	t.Run("handles EOF error with retry", func(t *testing.T) {
		retryClient := &mockIMAPClient{
			listFoldersResult: []*models.Folder{
				{Name: "INBOX", Role: "inbox"},
				{Name: "Drafts", Role: "drafts"},
			},
			listFoldersErr: nil,
		}

		rr, _ := testRetryScenario(t, pool, encryptor, email, userID,
			fmt.Errorf("failed to list folders: EOF"),
			retryClient)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 after retry, got %d", rr.Code)
		}
	})

	t.Run("returns error if retry also fails", func(t *testing.T) {
		retryClient := &mockIMAPClient{
			listFoldersResult: nil,
			listFoldersErr:    fmt.Errorf("failed to list folders: connection refused"),
		}

		rr, mockPool := testRetryScenario(t, pool, encryptor, email, userID,
			fmt.Errorf("failed to list folders: write: broken pipe"),
			retryClient)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 when retry also fails, got %d", rr.Code)
		}

		// Verify GetClient was called twice (initial plus retry)
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
		rr := callGetFolders(t, handler, email)

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

	t.Run("returns 400 when SPECIAL-USE not supported", func(t *testing.T) {
		mockClient := &mockIMAPClient{
			listFoldersResult: nil,
			listFoldersErr:    fmt.Errorf("IMAP server does not support SPECIAL-USE extension (RFC 6154), which is required for V-Mail to identify folder types"),
		}

		mockPool := &mockIMAPPool{
			getClientResult: mockClient,
			getClientErr:    nil,
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)
		rr := callGetFolders(t, handler, email)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}

		// Verify error message
		body := rr.Body.String()
		if !strings.Contains(body, "SPECIAL-USE") {
			t.Errorf("Expected error message to mention SPECIAL-USE, got: %s", body)
		}
	})

	t.Run("returns 500 when decrypting IMAP password fails", func(t *testing.T) {
		email := "decrypt-error@example.com"
		ctx := context.Background()

		// Create user
		userID, err := db.GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Create settings with corrupted encrypted password (invalid encrypted data)
		corruptedPassword := []byte("not-valid-encrypted-data")
		encryptedSMTPPassword, _ := encryptor.Encrypt("smtp_pass")

		settings := &models.UserSettings{
			UserID:                   userID,
			UndoSendDelaySeconds:     20,
			PaginationThreadsPerPage: 100,
			IMAPServerHostname:       "imap.test.com",
			IMAPUsername:             "user",
			EncryptedIMAPPassword:    corruptedPassword,
			SMTPServerHostname:       "smtp.test.com",
			SMTPUsername:             "user",
			EncryptedSMTPPassword:    encryptedSMTPPassword,
		}
		if err := db.SaveUserSettings(ctx, pool, settings); err != nil {
			t.Fatalf("Failed to save settings: %v", err)
		}

		handler := NewFoldersHandler(pool, encryptor, imap.NewPool())
		rr := callGetFolders(t, handler, email)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})

	t.Run("handles timeout error with StatusServiceUnavailable", func(t *testing.T) {
		email := "timeout-test@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		mockPool := &mockIMAPPool{
			getClientResult: nil,
			getClientErr:    fmt.Errorf("dial tcp 192.168.1.1:993: i/o timeout"),
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)
		rr := callGetFolders(t, handler, email)

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", rr.Code)
		}

		// Verify error message
		body := rr.Body.String()
		if !strings.Contains(body, "timed out") {
			t.Errorf("Expected error message to mention timeout, got: %s", body)
		}
		if !strings.Contains(body, "hostname") {
			t.Errorf("Expected error message to mention hostname, got: %s", body)
		}
	})
}

// failingResponseWriter is a ResponseWriter that fails on Write to test error handling.
type failingResponseWriter struct {
	http.ResponseWriter
	writeShouldFail bool
}

func (f *failingResponseWriter) Write(p []byte) (int, error) {
	if f.writeShouldFail {
		return 0, fmt.Errorf("write failed")
	}
	return f.ResponseWriter.Write(p)
}

func TestFoldersHandler_WriteResponseErrors(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	email := "write-error@example.com"
	setupTestUserAndSettings(t, pool, encryptor, email)

	t.Run("handles write failure gracefully", func(t *testing.T) {
		mockClient := &mockIMAPClient{
			listFoldersResult: []*models.Folder{
				{Name: "INBOX", Role: "inbox"},
			},
			listFoldersErr: nil,
		}

		mockPool := &mockIMAPPool{
			getClientResult: mockClient,
			getClientErr:    nil,
		}

		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		// Create a ResponseWriter that fails on Write
		rr := httptest.NewRecorder()
		failingWriter := &failingResponseWriter{
			ResponseWriter:  rr,
			writeShouldFail: true,
		}

		handler.GetFolders(failingWriter, req)

		// The handler should handle the write error gracefully (it logs but doesn't crash)
		// We can't easily test the error path without checking logs, but we verify it doesn't panic
	})
}

func TestSortFoldersByRole(t *testing.T) {
	tests := []struct {
		name     string
		folders  []*models.Folder
		expected []string // Expected folder names in order
	}{
		{
			name: "sorts by role priority",
			folders: []*models.Folder{
				{Name: "Archive", Role: "archive"},
				{Name: "INBOX", Role: "inbox"},
				{Name: "Drafts", Role: "drafts"},
				{Name: "Sent", Role: "sent"},
			},
			expected: []string{"INBOX", "Sent", "Drafts", "Archive"},
		},
		{
			name: "sorts alphabetically within same role",
			folders: []*models.Folder{
				{Name: "Zebra", Role: "other"},
				{Name: "Alpha", Role: "other"},
				{Name: "Beta", Role: "other"},
			},
			expected: []string{"Alpha", "Beta", "Zebra"},
		},
		{
			name: "sorts by role then alphabetically",
			folders: []*models.Folder{
				{Name: "Zebra", Role: "other"},
				{Name: "INBOX", Role: "inbox"},
				{Name: "Alpha", Role: "other"},
				{Name: "Sent", Role: "sent"},
				{Name: "Beta", Role: "other"},
			},
			expected: []string{"INBOX", "Sent", "Alpha", "Beta", "Zebra"},
		},
		{
			name: "handles all role types",
			folders: []*models.Folder{
				{Name: "Trash", Role: "trash"},
				{Name: "Spam", Role: "spam"},
				{Name: "INBOX", Role: "inbox"},
				{Name: "Sent", Role: "sent"},
				{Name: "Drafts", Role: "drafts"},
				{Name: "Archive", Role: "archive"},
			},
			expected: []string{"INBOX", "Sent", "Drafts", "Spam", "Trash", "Archive"},
		},
		{
			name:     "handles empty list",
			folders:  []*models.Folder{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original
			folders := make([]*models.Folder, len(tt.folders))
			for i, f := range tt.folders {
				folders[i] = &models.Folder{
					Name: f.Name,
					Role: f.Role,
				}
			}

			sortFoldersByRole(folders)

			if len(folders) != len(tt.expected) {
				t.Errorf("Expected %d folders, got %d", len(tt.expected), len(folders))
				return
			}

			for i, expectedName := range tt.expected {
				if folders[i].Name != expectedName {
					t.Errorf("Expected folder at index %d to be '%s', got '%s'", i, expectedName, folders[i].Name)
				}
			}
		})
	}
}
