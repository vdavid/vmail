package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
	ws "github.com/vdavid/vmail/backend/internal/websocket"
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

		// Mock IMAP service to return parser error wrapped with ErrInvalidSearchQuery
		mockIMAP.searchErr = fmt.Errorf("%w: empty from: value", imap.ErrInvalidSearchQuery)

		req := createRequestWithUser("GET", "/api/v1/search?q=from:", email)
		rr := httptest.NewRecorder()

		handler.Search(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("falls back to default limit when GetUserSettings returns error", func(t *testing.T) {
		email := "settings-error-search@example.com"
		ctx := context.Background()
		userID := setupTestUserAndSettings(t, pool, encryptor, email)

		// Delete the user settings to simulate GetUserSettings returning an error
		// (it will return NotFound, which getPaginationLimit handles by using default)
		if _, err := pool.Exec(ctx, "DELETE FROM user_settings WHERE user_id = $1", userID); err != nil {
			t.Fatalf("Failed to delete user settings: %v", err)
		}

		// Reset mock state
		mockIMAP.searchErr = nil
		mockIMAP.searchResult = []*models.Thread{}
		mockIMAP.searchCount = 0

		req := createRequestWithUser("GET", "/api/v1/search?q=test", email)
		rr := httptest.NewRecorder()

		handler.Search(rr, req)

		// Should still return 200 OK, using default limit of 100
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		// Verify that the search was called with default limit (100)
		// Since limitFromQuery is 0, it should use default
		if mockIMAP.searchLimit != 100 {
			t.Errorf("Expected default limit 100, got %d", mockIMAP.searchLimit)
		}
	})

	t.Run("handles JSON encoding failure gracefully", func(t *testing.T) {
		email := "json-error-search@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		// Reset mock state
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
		mockIMAP.searchErr = nil

		req := createRequestWithUser("GET", "/api/v1/search?q=test", email)

		// Create a ResponseWriter that fails on Write
		rr := httptest.NewRecorder()
		failingWriter := &failingResponseWriterSearch{
			ResponseWriter:  rr,
			writeShouldFail: true,
		}

		handler.Search(failingWriter, req)

		// The handler should handle the write error gracefully (it logs but doesn't crash)
		// The status code should still be set (200) even if Write fails
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	t.Run("handles invalid pagination parameters gracefully", func(t *testing.T) {
		email := "pagination-invalid-search@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		testCases := []struct {
			name          string
			query         string
			expectedPage  int
			expectedLimit int
		}{
			{"page=0 uses default", "q=test&page=0&limit=50", 1, 50},
			{"page=-1 uses default", "q=test&page=-1&limit=50", 1, 50},
			{"limit=0 uses default", "q=test&page=1&limit=0", 1, 100},
			{"limit=-1 uses default", "q=test&page=1&limit=-1", 1, 100},
			{"both invalid", "q=test&page=0&limit=0", 1, 100},
			{"non-numeric page", "q=test&page=abc&limit=50", 1, 50},
			{"non-numeric limit", "q=test&page=1&limit=xyz", 1, 100},
			{"very large limit", "q=test&page=1&limit=999999", 1, 999999},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Reset mock state for each test case
				mockIMAP.searchErr = nil
				mockIMAP.searchResult = []*models.Thread{}
				mockIMAP.searchCount = 0

				req := httptest.NewRequest("GET", "/api/v1/search?"+tc.query, nil)
				reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
				req = req.WithContext(reqCtx)

				rr := httptest.NewRecorder()
				handler.Search(rr, req)

				if rr.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", rr.Code)
				}

				if mockIMAP.searchPage != tc.expectedPage {
					t.Errorf("Expected page %d, got %d", tc.expectedPage, mockIMAP.searchPage)
				}
				if mockIMAP.searchLimit != tc.expectedLimit {
					t.Errorf("Expected limit %d, got %d", tc.expectedLimit, mockIMAP.searchLimit)
				}
			})
		}
	})
}

// failingResponseWriterSearch is a ResponseWriter that fails on Write to test error handling.
type failingResponseWriterSearch struct {
	http.ResponseWriter
	writeShouldFail bool
}

func (f *failingResponseWriterSearch) Write(p []byte) (int, error) {
	if f.writeShouldFail {
		return 0, fmt.Errorf("write failed")
	}
	return f.ResponseWriter.Write(p)
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

// StartIdleListener is part of the IMAPService interface but is not used in search tests.
func (m *mockIMAPServiceForSearch) StartIdleListener(context.Context, string, *ws.Hub) {
}

type imapError struct {
	message string
}

func (e *imapError) Error() string {
	return e.message
}
