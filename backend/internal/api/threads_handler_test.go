package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
	"github.com/vdavid/vmail/backend/internal/testutil/mocks"
)

func TestThreadsHandler_GetThreads(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()
	encryptor := getTestEncryptor(t)

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		mockService := mocks.NewIMAPService(t)
		handler := NewThreadsHandler(pool, encryptor, mockService)
		req := httptest.NewRequest("GET", "/api/v1/threads?folder=INBOX", nil)
		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("returns 400 when folder parameter is missing", func(t *testing.T) {
		mockService := mocks.NewIMAPService(t)
		handler := NewThreadsHandler(pool, encryptor, mockService)
		req := createRequestWithUser("GET", "/api/v1/threads", "user@example.com")
		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("GetThreads scenarios", func(t *testing.T) {
		// We'll use a single user/setup for simplicity where possible, or unique per test
		tests := []struct {
			name           string
			query          string
			setupData      func(context.Context, string)    // userID
			setupMock      func(*mocks.IMAPService, string) // userID
			expectedStatus int
			expectedBody   string
			checkResponse  func(*testing.T, *httptest.ResponseRecorder)
		}{
			{
				name:  "returns empty list when no threads exist",
				query: "folder=INBOX",
				setupMock: func(ms *mocks.IMAPService, userID string) {
					ms.On("ShouldSyncFolder", mock.Anything, userID, "INBOX").Return(false, nil)
				},
				expectedStatus: http.StatusOK,
				checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
					var response models.ThreadsResponse
					err := json.NewDecoder(rr.Body).Decode(&response)
					assert.NoError(t, err)
					assert.Empty(t, response.Threads)
					assert.Equal(t, 0, response.Pagination.TotalCount)
				},
			},
			{
				name:  "returns threads from database",
				query: "folder=INBOX",
				setupData: func(ctx context.Context, userID string) {
					thread := &models.Thread{
						UserID:         userID,
						StableThreadID: "test-thread-123",
						Subject:        "Test Thread",
					}
					assert.NoError(t, db.SaveThread(ctx, pool, thread))
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
					assert.NoError(t, db.SaveMessage(ctx, pool, msg))
				},
				setupMock: func(ms *mocks.IMAPService, userID string) {
					ms.On("ShouldSyncFolder", mock.Anything, userID, "INBOX").Return(false, nil)
				},
				expectedStatus: http.StatusOK,
				checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
					var response models.ThreadsResponse
					err := json.NewDecoder(rr.Body).Decode(&response)
					assert.NoError(t, err)
					assert.Len(t, response.Threads, 1)
					assert.Equal(t, "test-thread-123", response.Threads[0].StableThreadID)
				},
			},
			{
				name:  "calls SyncThreadsForFolder when cache is stale",
				query: "folder=INBOX",
				setupMock: func(ms *mocks.IMAPService, userID string) {
					ms.On("ShouldSyncFolder", mock.Anything, userID, "INBOX").Return(true, nil)
					ms.On("SyncThreadsForFolder", mock.Anything, userID, "INBOX").Return(nil)
				},
				expectedStatus: http.StatusOK,
			},
			{
				name:  "continues even if sync fails",
				query: "folder=INBOX",
				setupMock: func(ms *mocks.IMAPService, userID string) {
					ms.On("ShouldSyncFolder", mock.Anything, userID, "INBOX").Return(true, nil)
					ms.On("SyncThreadsForFolder", mock.Anything, userID, "INBOX").Return(fmt.Errorf("IMAP connection failed"))
				},
				expectedStatus: http.StatusOK,
			},
			{
				name:  "continues when ShouldSyncFolder returns an error",
				query: "folder=INBOX",
				setupMock: func(ms *mocks.IMAPService, userID string) {
					ms.On("ShouldSyncFolder", mock.Anything, userID, "INBOX").Return(true, fmt.Errorf("cache check failed"))
					ms.On("SyncThreadsForFolder", mock.Anything, userID, "INBOX").Return(nil)
				},
				expectedStatus: http.StatusOK,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Unique user for each test to avoid data pollution
				// Using t.Name() as unique identifier (sanitized)
				uniqueName := fmt.Sprintf("user-%d", time.Now().UnixNano())
				email := fmt.Sprintf("%s@example.com", uniqueName)
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				if tt.setupData != nil {
					tt.setupData(context.Background(), userID)
				}

				mockService := mocks.NewIMAPService(t)
				if tt.setupMock != nil {
					tt.setupMock(mockService, userID)
				}

				handler := NewThreadsHandler(pool, encryptor, mockService)
				req := createRequestWithUser("GET", "/api/v1/threads?"+tt.query, email)
				rr := httptest.NewRecorder()
				handler.GetThreads(rr, req)

				assert.Equal(t, tt.expectedStatus, rr.Code)
				if tt.expectedBody != "" {
					assert.Contains(t, rr.Body.String(), tt.expectedBody)
				}
				if tt.checkResponse != nil {
					tt.checkResponse(t, rr)
				}
			})
		}
	})

	t.Run("returns 500 when GetThreadsForFolder returns an error", func(t *testing.T) {
		email := "threads-error@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		// Use a canceled context to simulate database error
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		mockService := mocks.NewIMAPService(t)
		// Mock won't be called because context cancellation causes DB error first
		// But if it were, it would look like this:
		// mockService.On("ShouldSyncFolder", ...).Return(false, nil)

		handler := NewThreadsHandler(pool, encryptor, mockService)
		req := httptest.NewRequest("GET", "/api/v1/threads?folder=INBOX", nil)
		reqCtx := context.WithValue(canceledCtx, auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("handles JSON encoding failure gracefully", func(t *testing.T) {
		email := "json-error@example.com"
		userID := setupTestUserAndSettings(t, pool, encryptor, email)
		ctx := context.Background()

		thread := &models.Thread{
			UserID:         userID,
			StableThreadID: "test-thread-json-error",
			Subject:        "Test Thread",
		}
		assert.NoError(t, db.SaveThread(ctx, pool, thread))

		mockService := mocks.NewIMAPService(t)
		mockService.On("ShouldSyncFolder", mock.Anything, userID, "INBOX").Return(false, nil)

		handler := NewThreadsHandler(pool, encryptor, mockService)
		req := createRequestWithUser("GET", "/api/v1/threads?folder=INBOX", email)
		rr := httptest.NewRecorder()
		failingWriter := &FailingResponseWriter{
			ResponseWriter:  rr,
			WriteShouldFail: true,
		}

		handler.GetThreads(failingWriter, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("falls back to default limit when GetUserSettings fails", func(t *testing.T) {
		email := "settings-error@example.com"
		ctx := context.Background()
		userID := setupTestUserAndSettings(t, pool, encryptor, email)

		// Delete the user settings to simulate GetUserSettings returning an error
		_, err := pool.Exec(ctx, "DELETE FROM user_settings WHERE user_id = $1", userID)
		assert.NoError(t, err)

		mockService := mocks.NewIMAPService(t)
		mockService.On("ShouldSyncFolder", mock.Anything, userID, "INBOX").Return(false, nil)

		handler := NewThreadsHandler(pool, encryptor, mockService)
		req := createRequestWithUser("GET", "/api/v1/threads?folder=INBOX", email)
		rr := httptest.NewRecorder()
		handler.GetThreads(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var response models.ThreadsResponse
		json.NewDecoder(rr.Body).Decode(&response)
		assert.Equal(t, 100, response.Pagination.PerPage)
	})
}
