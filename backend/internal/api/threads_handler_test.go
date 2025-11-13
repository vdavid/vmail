package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestThreadsHandler_GetThreads(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	imapService := imap.NewService(pool, encryptor)
	handler := NewThreadsHandler(pool, encryptor, imapService)

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/threads?folder=INBOX", nil)

		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("returns 400 when folder parameter is missing", func(t *testing.T) {
		email := "user@example.com"

		req := httptest.NewRequest("GET", "/api/v1/threads", nil)
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("returns empty list when no threads exist", func(t *testing.T) {
		email := "user@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		req := createRequestWithUser("GET", "/api/v1/threads?folder=INBOX", email)
		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response struct {
			Threads    []*models.Thread `json:"threads"`
			Pagination struct {
				TotalCount int `json:"total_count"`
				Page       int `json:"page"`
				PerPage    int `json:"per_page"`
			} `json:"pagination"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Threads) != 0 {
			t.Errorf("Expected empty threads list, got %d threads", len(response.Threads))
		}
		if response.Pagination.TotalCount != 0 {
			t.Errorf("Expected total_count 0, got %d", response.Pagination.TotalCount)
		}
	})

	t.Run("returns threads from database", func(t *testing.T) {
		email := "threaduser@example.com"
		ctx := context.Background()
		userID := setupTestUserAndSettings(t, pool, encryptor, email)

		// Create a thread with messages
		thread := &models.Thread{
			UserID:         userID,
			StableThreadID: "test-thread-123",
			Subject:        "Test Thread",
		}
		if err := db.SaveThread(ctx, pool, thread); err != nil {
			t.Fatalf("Failed to save thread: %v", err)
		}

		now := time.Now()
		msg := &models.Message{
			ThreadID:        thread.ID,
			UserID:          userID,
			IMAPUID:         1,
			IMAPFolderName:  "INBOX",
			MessageIDHeader: "msg-123",
			Subject:         "Test Thread",
			SentAt:          &now,
		}
		if err := db.SaveMessage(ctx, pool, msg); err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/v1/threads?folder=INBOX", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response struct {
			Threads    []*models.Thread `json:"threads"`
			Pagination struct {
				TotalCount int `json:"total_count"`
				Page       int `json:"page"`
				PerPage    int `json:"per_page"`
			} `json:"pagination"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Threads) != 1 {
			t.Errorf("Expected 1 thread, got %d", len(response.Threads))
		}

		if response.Threads[0].StableThreadID != "test-thread-123" {
			t.Errorf("Expected thread ID 'test-thread-123', got %s", response.Threads[0].StableThreadID)
		}
		if response.Pagination.TotalCount != 1 {
			t.Errorf("Expected total_count 1, got %d", response.Pagination.TotalCount)
		}
	})

	t.Run("respects pagination parameters", func(t *testing.T) {
		email := "paginationuser@example.com"
		ctx := context.Background()
		userID := setupTestUserAndSettings(t, pool, encryptor, email)

		// Create multiple threads
		for i := 0; i < 3; i++ {
			threadID := fmt.Sprintf("thread-%d", i)
			thread := &models.Thread{
				UserID:         userID,
				StableThreadID: threadID,
				Subject:        fmt.Sprintf("Thread %d", i),
			}
			if err := db.SaveThread(ctx, pool, thread); err != nil {
				t.Fatalf("Failed to save thread: %v", err)
			}

			now := time.Now()
			msgID := fmt.Sprintf("msg-%d", i)
			msg := &models.Message{
				ThreadID:        thread.ID,
				UserID:          userID,
				IMAPUID:         int64(i + 1),
				IMAPFolderName:  "INBOX",
				MessageIDHeader: msgID,
				Subject:         fmt.Sprintf("Thread %d", i),
				SentAt:          &now,
			}
			if err := db.SaveMessage(ctx, pool, msg); err != nil {
				t.Fatalf("Failed to save message: %v", err)
			}
		}

		req := createRequestWithUser("GET", "/api/v1/threads?folder=INBOX&page=1&limit=2", email)

		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response struct {
			Threads    []*models.Thread `json:"threads"`
			Pagination struct {
				TotalCount int `json:"total_count"`
				Page       int `json:"page"`
				PerPage    int `json:"per_page"`
			} `json:"pagination"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response.Threads) > 2 {
			t.Errorf("Expected at most 2 threads with limit, got %d", len(response.Threads))
		}
		if response.Pagination.Page != 1 {
			t.Errorf("Expected page 1, got %d", response.Pagination.Page)
		}
		if response.Pagination.PerPage != 2 {
			t.Errorf("Expected per_page 2, got %d", response.Pagination.PerPage)
		}
		if response.Pagination.TotalCount != 3 {
			t.Errorf("Expected total_count 3, got %d", response.Pagination.TotalCount)
		}

		// Test page 2
		req2 := createRequestWithUser("GET", "/api/v1/threads?folder=INBOX&page=2&limit=2", email)

		rr2 := httptest.NewRecorder()
		handler.GetThreads(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr2.Code)
		}

		var response2 struct {
			Threads    []*models.Thread `json:"threads"`
			Pagination struct {
				TotalCount int `json:"total_count"`
				Page       int `json:"page"`
				PerPage    int `json:"per_page"`
			} `json:"pagination"`
		}
		if err := json.NewDecoder(rr2.Body).Decode(&response2); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response2.Threads) > 1 {
			t.Errorf("Expected at most 1 thread on page 2, got %d", len(response2.Threads))
		}
		if response2.Pagination.Page != 2 {
			t.Errorf("Expected page 2, got %d", response2.Pagination.Page)
		}
		if response2.Pagination.TotalCount != 3 {
			t.Errorf("Expected total_count 3, got %d", response2.Pagination.TotalCount)
		}
	})
}

// mockIMAPService is a mock implementation of IMAPService for testing
type mockIMAPService struct {
	shouldSyncFolderResult     bool
	shouldSyncFolderErr        error
	syncThreadsForFolderErr    error
	shouldSyncFolderCalled     bool
	syncThreadsForFolderCalled bool
	syncThreadsForFolderUserID string
	syncThreadsForFolderFolder string
}

func (m *mockIMAPService) ShouldSyncFolder(context.Context, string, string) (bool, error) {
	m.shouldSyncFolderCalled = true
	return m.shouldSyncFolderResult, m.shouldSyncFolderErr
}

func (m *mockIMAPService) SyncThreadsForFolder(_ context.Context, userID, folderName string) error {
	m.syncThreadsForFolderCalled = true
	m.syncThreadsForFolderUserID = userID
	m.syncThreadsForFolderFolder = folderName
	return m.syncThreadsForFolderErr
}

func (m *mockIMAPService) SyncFullMessage(context.Context, string, string, int64) error {
	return nil
}

func (m *mockIMAPService) SyncFullMessages(context.Context, string, []imap.MessageToSync) error {
	return nil
}

func (m *mockIMAPService) Search(context.Context, string, string, int, int) ([]*models.Thread, int, error) {
	return nil, 0, nil
}

func (m *mockIMAPService) Close() {}

func TestThreadsHandler_SyncsWhenStale(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	email := "sync-test@example.com"
	userID := setupTestUserAndSettings(t, pool, encryptor, email)

	t.Run("calls SyncThreadsForFolder when cache is stale", func(t *testing.T) {
		mockIMAP := &mockIMAPService{
			shouldSyncFolderResult:  true, // Cache is stale
			shouldSyncFolderErr:     nil,
			syncThreadsForFolderErr: nil, // Sync succeeds
		}

		handler := NewThreadsHandler(pool, encryptor, mockIMAP)
		req := createRequestWithUser("GET", "/api/v1/threads?folder=INBOX", email)

		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Verify that ShouldSyncFolder was called
		if !mockIMAP.shouldSyncFolderCalled {
			t.Error("Expected ShouldSyncFolder to be called")
		}

		// Verify that SyncThreadsForFolder was called
		if !mockIMAP.syncThreadsForFolderCalled {
			t.Error("Expected SyncThreadsForFolder to be called when cache is stale")
		}

		// Verify correct parameters were passed
		if mockIMAP.syncThreadsForFolderUserID != userID {
			t.Errorf("Expected SyncThreadsForFolder to be called with userID %s, got %s", userID, mockIMAP.syncThreadsForFolderUserID)
		}
		if mockIMAP.syncThreadsForFolderFolder != "INBOX" {
			t.Errorf("Expected SyncThreadsForFolder to be called with folder 'INBOX', got %s", mockIMAP.syncThreadsForFolderFolder)
		}
	})

	t.Run("does not call SyncThreadsForFolder when cache is fresh", func(t *testing.T) {
		mockIMAP := &mockIMAPService{
			shouldSyncFolderResult:  false, // Cache is fresh
			shouldSyncFolderErr:     nil,
			syncThreadsForFolderErr: nil,
		}

		handler := NewThreadsHandler(pool, encryptor, mockIMAP)
		req := createRequestWithUser("GET", "/api/v1/threads?folder=INBOX", email)

		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Verify that we called ShouldSyncFolder
		if !mockIMAP.shouldSyncFolderCalled {
			t.Error("Expected ShouldSyncFolder to be called")
		}

		// Verify that we did NOT call SyncThreadsForFolder
		if mockIMAP.syncThreadsForFolderCalled {
			t.Error("Expected SyncThreadsForFolder NOT to be called when cache is fresh")
		}
	})

	t.Run("continues even if sync fails", func(t *testing.T) {
		mockIMAP := &mockIMAPService{
			shouldSyncFolderResult:  true,
			shouldSyncFolderErr:     nil,
			syncThreadsForFolderErr: fmt.Errorf("IMAP connection failed"), // Sync fails
		}

		handler := NewThreadsHandler(pool, encryptor, mockIMAP)
		req := createRequestWithUser("GET", "/api/v1/threads?folder=INBOX", email)

		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		// Handler should still return 200 OK even if sync fails
		// It falls back to cached data
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 even when sync fails, got %d", rr.Code)
		}

		// Verify sync was attempted
		if !mockIMAP.syncThreadsForFolderCalled {
			t.Error("Expected SyncThreadsForFolder to be called even if it fails")
		}
	})
}
