package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
	"github.com/vdavid/vmail/backend/internal/testutil/mocks"
)

func TestSearchHandler_Search(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()
	encryptor := getTestEncryptor(t)

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		mockService := mocks.NewIMAPService(t)
		handler := NewSearchHandler(pool, encryptor, mockService)
		VerifyAuthCheck(t, handler.Search, "GET", "/api/v1/search?q=test")
	})

	t.Run("Search scenarios", func(t *testing.T) {
		tests := []struct {
			name           string
			query          string
			setupMock      func(*mocks.IMAPService)
			setupUser      bool
			expectedStatus int
			expectedBody   string
			checkMock      func(*testing.T, *mocks.IMAPService)
		}{
			{
				name:      "calls imapService.Search with correct params",
				query:     "q=test&page=2&limit=50",
				setupUser: true,
				setupMock: func(ms *mocks.IMAPService) {
					ms.On("Search", mock.Anything, mock.Anything, "test", 2, 50).
						Return([]*models.Thread{}, 0, nil)
				},
				expectedStatus: http.StatusOK,
			},
			{
				name:      "handles empty query",
				query:     "q=",
				setupUser: true,
				setupMock: func(ms *mocks.IMAPService) {
					ms.On("Search", mock.Anything, mock.Anything, "", 1, 100).
						Return([]*models.Thread{}, 0, nil)
				},
				expectedStatus: http.StatusOK,
			},
			{
				name:      "returns correct JSON response",
				query:     "q=test",
				setupUser: true,
				setupMock: func(ms *mocks.IMAPService) {
					threads := []*models.Thread{
						{
							ID:             "thread-1",
							StableThreadID: "stable-1",
							Subject:        "Test Thread",
							UserID:         "user-1",
						},
					}
					ms.On("Search", mock.Anything, mock.Anything, "test", 1, 100).
						Return(threads, 1, nil)
				},
				expectedStatus: http.StatusOK,
				expectedBody:   "thread-1",
			},
			{
				name:      "handles IMAP service errors",
				query:     "q=test",
				setupUser: true,
				setupMock: func(ms *mocks.IMAPService) {
					ms.On("Search", mock.Anything, mock.Anything, "test", 1, 100).
						Return(nil, 0, fmt.Errorf("IMAP connection failed"))
				},
				expectedStatus: http.StatusInternalServerError,
			},
			{
				name:      "returns 400 for invalid query syntax",
				query:     "q=from:",
				setupUser: true,
				setupMock: func(ms *mocks.IMAPService) {
					ms.On("Search", mock.Anything, mock.Anything, "from:", 1, 100).
						Return(nil, 0, fmt.Errorf("%w: empty from: value", imap.ErrInvalidSearchQuery))
				},
				expectedStatus: http.StatusBadRequest,
			},
			// Pagination tests
			{
				name:      "page=0 uses default",
				query:     "q=test&page=0&limit=50",
				setupUser: true,
				setupMock: func(ms *mocks.IMAPService) {
					ms.On("Search", mock.Anything, mock.Anything, "test", 1, 50).Return([]*models.Thread{}, 0, nil)
				},
				expectedStatus: http.StatusOK,
			},
			{
				name:      "limit=0 uses default",
				query:     "q=test&page=1&limit=0",
				setupUser: true,
				setupMock: func(ms *mocks.IMAPService) {
					ms.On("Search", mock.Anything, mock.Anything, "test", 1, 100).Return([]*models.Thread{}, 0, nil)
				},
				expectedStatus: http.StatusOK,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				email := fmt.Sprintf("search-%s@example.com", strings.ReplaceAll(tt.name, " ", "-"))
				if tt.setupUser {
					setupTestUserAndSettings(t, pool, encryptor, email)
				}

				mockService := mocks.NewIMAPService(t)
				if tt.setupMock != nil {
					tt.setupMock(mockService)
				}

				handler := NewSearchHandler(pool, encryptor, mockService)
				req := createRequestWithUser("GET", "/api/v1/search?"+tt.query, email)
				rr := httptest.NewRecorder()

				handler.Search(rr, req)

				assert.Equal(t, tt.expectedStatus, rr.Code)
				if tt.expectedBody != "" {
					assert.Contains(t, rr.Body.String(), tt.expectedBody)
				}
				if tt.checkMock != nil {
					tt.checkMock(t, mockService)
				}
			})
		}
	})

	t.Run("falls back to default limit when GetUserSettings returns error", func(t *testing.T) {
		email := "settings-error-search@example.com"
		ctx := context.Background()
		userID := setupTestUserAndSettings(t, pool, encryptor, email)

		// Delete the user settings to simulate GetUserSettings returning an error
		// (it will return NotFound, which getPaginationLimit handles by using default)
		_, err := pool.Exec(ctx, "DELETE FROM user_settings WHERE user_id = $1", userID)
		assert.NoError(t, err)

		mockService := mocks.NewIMAPService(t)
		mockService.On("Search", mock.Anything, mock.Anything, "test", 1, 100).Return([]*models.Thread{}, 0, nil)

		handler := NewSearchHandler(pool, encryptor, mockService)
		req := createRequestWithUser("GET", "/api/v1/search?q=test", email)
		rr := httptest.NewRecorder()
		handler.Search(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("handles JSON encoding failure gracefully", func(t *testing.T) {
		email := "json-error-search@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		mockService := mocks.NewIMAPService(t)
		threads := []*models.Thread{
			{
				ID:             "thread-1",
				StableThreadID: "stable-1",
				Subject:        "Test Thread",
				UserID:         "user-1",
			},
		}
		mockService.On("Search", mock.Anything, mock.Anything, "test", 1, 100).
			Return(threads, 1, nil)

		handler := NewSearchHandler(pool, encryptor, mockService)
		req := createRequestWithUser("GET", "/api/v1/search?q=test", email)
		rr := httptest.NewRecorder()
		failingWriter := &FailingResponseWriter{
			ResponseWriter:  rr,
			WriteShouldFail: true,
		}

		handler.Search(failingWriter, req)

		// The handler should handle the write error gracefully (it logs but doesn't crash)
		// The status code should still be set (200) even if Write fails
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
