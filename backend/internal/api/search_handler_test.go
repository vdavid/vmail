package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestSearchHandler_Search(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	mockIMAP := &mockIMAPServiceForSearch{
		searchResult: []*models.Thread{},
		searchCount:  0,
		searchErr:    nil,
	}
	handler := NewSearchHandler(pool, encryptor, mockIMAP)

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/search?q=test", nil)
		rr := httptest.NewRecorder()
		handler.Search(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("calls imapService.Search with correct params", func(t *testing.T) {
		email := "searchuser@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		req := createRequestWithUser("GET", "/api/v1/search?q=test&page=2&limit=50", email)
		rr := httptest.NewRecorder()

		handler.Search(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if mockIMAP.searchQuery != "test" {
			t.Errorf("Expected query 'test', got '%s'", mockIMAP.searchQuery)
		}
		if mockIMAP.searchPage != 2 {
			t.Errorf("Expected page 2, got %d", mockIMAP.searchPage)
		}
		if mockIMAP.searchLimit != 50 {
			t.Errorf("Expected limit 50, got %d", mockIMAP.searchLimit)
		}
	})

	t.Run("handles empty query", func(t *testing.T) {
		email := "searchuser2@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		req := createRequestWithUser("GET", "/api/v1/search?q=", email)
		rr := httptest.NewRecorder()

		handler.Search(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if mockIMAP.searchQuery != "" {
			t.Errorf("Expected empty query, got '%s'", mockIMAP.searchQuery)
		}
	})

	t.Run("returns correct JSON response", func(t *testing.T) {
		email := "searchuser3@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		threads := []*models.Thread{
			{
				ID:             "thread-1",
				StableThreadID: "stable-1",
				Subject:        "Test Thread",
				UserID:         "user-1",
			},
		}
		mockIMAP.searchResult = threads
		mockIMAP.searchCount = 1

		req := createRequestWithUser("GET", "/api/v1/search?q=test", email)
		rr := httptest.NewRecorder()

		handler.Search(rr, req)

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
		if response.Pagination.TotalCount != 1 {
			t.Errorf("Expected total_count 1, got %d", response.Pagination.TotalCount)
		}
	})

	t.Run("handles IMAP service errors", func(t *testing.T) {
		email := "searchuser4@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		mockIMAP.searchErr = &imapError{message: "IMAP connection failed"}

		req := createRequestWithUser("GET", "/api/v1/search?q=test", email)
		rr := httptest.NewRecorder()

		handler.Search(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})

	t.Run("returns 400 for invalid query syntax", func(t *testing.T) {
		email := "searchuser5@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		// Mock IMAP service to return parser error
		mockIMAP.searchErr = &imapError{message: "invalid search query: empty from: value"}

		req := createRequestWithUser("GET", "/api/v1/search?q=from:", email)
		rr := httptest.NewRecorder()

		handler.Search(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})
}

// mockIMAPServiceForSearch is a mock implementation of IMAPService for search tests
type mockIMAPServiceForSearch struct {
	searchQuery  string
	searchPage   int
	searchLimit  int
	searchResult []*models.Thread
	searchCount  int
	searchErr    error
}

func (m *mockIMAPServiceForSearch) ShouldSyncFolder(context.Context, string, string) (bool, error) {
	return false, nil
}

func (m *mockIMAPServiceForSearch) SyncThreadsForFolder(context.Context, string, string) error {
	return nil
}

func (m *mockIMAPServiceForSearch) SyncFullMessage(context.Context, string, string, int64) error {
	return nil
}

func (m *mockIMAPServiceForSearch) SyncFullMessages(context.Context, string, []imap.MessageToSync) error {
	return nil
}

func (m *mockIMAPServiceForSearch) Search(_ context.Context, _ string, query string, page, limit int) ([]*models.Thread, int, error) {
	m.searchQuery = query
	m.searchPage = page
	m.searchLimit = limit
	return m.searchResult, m.searchCount, m.searchErr
}

func (m *mockIMAPServiceForSearch) Close() {}

type imapError struct {
	message string
}

func (e *imapError) Error() string {
	return e.message
}
