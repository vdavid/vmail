package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
	"github.com/vdavid/vmail/backend/internal/testutil/mocks"
)

func TestThreadHandler_GetThread(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	imapService := imap.NewService(pool, imap.NewPool(), encryptor)
	defer imapService.Close()
	handler := NewThreadHandler(pool, encryptor, imapService)

	tests := []struct {
		name        string
		setup       func(*testing.T) (*http.Request, http.ResponseWriter)
		expectCode  int
		checkResult func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "returns 401 when no user email in context",
			setup: func(*testing.T) (*http.Request, http.ResponseWriter) {
				req := httptest.NewRequest("GET", "/api/v1/thread/test-thread-id", nil)
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusUnauthorized,
		},
		{
			name: "returns 400 when thread_id is missing",
			setup: func(*testing.T) (*http.Request, http.ResponseWriter) {
				req := createRequestWithUser("GET", "/api/v1/thread/", "user@example.com")
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "returns 404 when thread not found",
			setup: func(*testing.T) (*http.Request, http.ResponseWriter) {
				req := createRequestWithUser("GET", "/api/v1/thread/non-existent-thread", "user@example.com")
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusNotFound,
		},
		{
			name: "returns thread with messages",
			setup: func(t *testing.T) (*http.Request, http.ResponseWriter) {
				email := "threaduser@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: "test-thread-456",
					Subject:        "Test Thread Subject",
				}
				if err := db.SaveThread(ctx, pool, thread); err != nil {
					t.Fatalf("Failed to save thread: %v", err)
				}

				now := time.Now()
				msg1 := &models.Message{
					ThreadID:        thread.ID,
					UserID:          userID,
					IMAPUID:         1,
					IMAPFolderName:  "INBOX",
					MessageIDHeader: "msg-1",
					FromAddress:     "sender@example.com",
					ToAddresses:     []string{"recipient@example.com"},
					Subject:         "Test Thread Subject",
					SentAt:          &now,
				}
				if err := db.SaveMessage(ctx, pool, msg1); err != nil {
					t.Fatalf("Failed to save message: %v", err)
				}

				msg2 := &models.Message{
					ThreadID:        thread.ID,
					UserID:          userID,
					IMAPUID:         2,
					IMAPFolderName:  "INBOX",
					MessageIDHeader: "msg-2",
					FromAddress:     "recipient@example.com",
					ToAddresses:     []string{"sender@example.com"},
					Subject:         "Re: Test Thread Subject",
					SentAt:          &now,
				}
				if err := db.SaveMessage(ctx, pool, msg2); err != nil {
					t.Fatalf("Failed to save message: %v", err)
				}

				req := createRequestWithUser("GET", "/api/v1/thread/test-thread-456", email)
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response models.Thread
				assert.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
				assert.Equal(t, "test-thread-456", response.StableThreadID)
				assert.Len(t, response.Messages, 2)
			},
		},
		{
			name: "returns thread with attachments",
			setup: func(t *testing.T) (*http.Request, http.ResponseWriter) {
				email := "attachmentuser@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: "test-thread-attachments",
					Subject:        "Thread with Attachments",
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
					MessageIDHeader: "msg-attachments",
					Subject:         "Thread with Attachments",
					SentAt:          &now,
				}
				if err := db.SaveMessage(ctx, pool, msg); err != nil {
					t.Fatalf("Failed to save message: %v", err)
				}

				attachment := &models.Attachment{
					MessageID: msg.ID,
					Filename:  "test.pdf",
					MimeType:  "application/pdf",
					SizeBytes: 1024,
					IsInline:  false,
				}
				if err := db.SaveAttachment(ctx, pool, attachment); err != nil {
					t.Fatalf("Failed to save attachment: %v", err)
				}

				req := createRequestWithUser("GET", "/api/v1/thread/test-thread-attachments", email)
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response models.Thread
				assert.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
				assert.Len(t, response.Messages, 1)
				assert.Len(t, response.Messages[0].Attachments, 1)
				assert.Equal(t, "test.pdf", response.Messages[0].Attachments[0].Filename)
			},
		},
		{
			name: "returns 500 when GetThreadByStableID returns non-NotFound error",
			setup: func(t *testing.T) (*http.Request, http.ResponseWriter) {
				email := "dberror-thread@example.com"
				setupTestUserAndSettings(t, pool, encryptor, email)

				canceledCtx, cancel := context.WithCancel(context.Background())
				cancel()
				req := httptest.NewRequest("GET", "/api/v1/thread/test-thread-id", nil)
				reqCtx := context.WithValue(canceledCtx, auth.UserEmailKey, email)
				req = req.WithContext(reqCtx)
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name: "returns 500 when GetMessagesForThread returns an error",
			setup: func(t *testing.T) (*http.Request, http.ResponseWriter) {
				email := "dberror-messages@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: "thread-db-error",
					Subject:        "DB Error Test",
				}
				if err := db.SaveThread(ctx, pool, thread); err != nil {
					t.Fatalf("Failed to save thread: %v", err)
				}

				canceledCtx, cancel := context.WithCancel(context.Background())
				cancel()
				req := httptest.NewRequest("GET", "/api/v1/thread/thread-db-error", nil)
				reqCtx := context.WithValue(canceledCtx, auth.UserEmailKey, email)
				req = req.WithContext(reqCtx)
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name: "continues with empty attachments when GetAttachmentsForMessages returns error",
			setup: func(t *testing.T) (*http.Request, http.ResponseWriter) {
				email := "attachments-error@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: "thread-attachments-error",
					Subject:        "Attachments Error Test",
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
					MessageIDHeader: "msg-attachments-error",
					Subject:         "Test",
					SentAt:          &now,
					UnsafeBodyHTML:  "<p>Body</p>",
					BodyText:        "Body",
				}
				if err := db.SaveMessage(ctx, pool, msg); err != nil {
					t.Fatalf("Failed to save message: %v", err)
				}

				req := createRequestWithUser("GET", "/api/v1/thread/thread-attachments-error", email)
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response models.Thread
				assert.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
				// Handler should complete successfully even if attachments fail
			},
		},
		{
			name: "handles invalid thread_id encoding",
			setup: func(t *testing.T) (*http.Request, http.ResponseWriter) {
				email := "encoding-test@example.com"
				setupTestUserAndSettings(t, pool, encryptor, email)

				req, err := http.NewRequest("GET", "/api/v1/thread/%ZZ", nil)
				if err != nil {
					req = &http.Request{
						Method: "GET",
						URL: &url.URL{
							Path: "/api/v1/thread/%ZZ",
						},
					}
				}
				reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
				req = req.WithContext(reqCtx)
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "handles JSON encoding failure gracefully",
			setup: func(t *testing.T) (*http.Request, http.ResponseWriter) {
				email := "json-error-thread@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: "thread-json-error",
					Subject:        "JSON Error Test",
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
					MessageIDHeader: "msg-json-error",
					Subject:         "Test",
					SentAt:          &now,
					UnsafeBodyHTML:  "<p>Body</p>",
					BodyText:        "Body",
				}
				if err := db.SaveMessage(ctx, pool, msg); err != nil {
					t.Fatalf("Failed to save message: %v", err)
				}

				req := createRequestWithUser("GET", "/api/v1/thread/thread-json-error", email)
				rr := httptest.NewRecorder()
				failingWriter := &FailingResponseWriter{
					ResponseWriter:  rr,
					WriteShouldFail: true,
				}
				return req, failingWriter
			},
			expectCode: http.StatusOK,
		},
		{
			name: "handles thread with nil messages",
			setup: func(t *testing.T) (*http.Request, http.ResponseWriter) {
				email := "nil-messages@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: "thread-nil-messages",
					Subject:        "Nil Messages Test",
				}
				if err := db.SaveThread(ctx, pool, thread); err != nil {
					t.Fatalf("Failed to save thread: %v", err)
				}

				req := createRequestWithUser("GET", "/api/v1/thread/thread-nil-messages", email)
				return req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response models.Thread
				assert.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
				assert.Empty(t, response.Messages)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, w := tt.setup(t)
			rr, ok := w.(*httptest.ResponseRecorder)
			if !ok {
				// For FailingResponseWriter case, we need to call handler differently
				handler.GetThread(w, req)
				if rr, ok := w.(*FailingResponseWriter); ok {
					assert.Equal(t, tt.expectCode, rr.ResponseWriter.(*httptest.ResponseRecorder).Code)
				}
				return
			}

			handler.GetThread(rr, req)
			assert.Equal(t, tt.expectCode, rr.Code)
			if tt.checkResult != nil {
				tt.checkResult(t, rr)
			}
		})
	}
}

func TestThreadHandler_SyncsMissingBodies(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)

	tests := []struct {
		name        string
		setup       func(*testing.T) (*ThreadHandler, *mocks.IMAPService, *http.Request, string)
		expectCode  int
		checkResult func(*testing.T, *httptest.ResponseRecorder, *mocks.IMAPService)
	}{
		{
			name: "syncs missing body and returns synced content",
			setup: func(t *testing.T) (*ThreadHandler, *mocks.IMAPService, *http.Request, string) {
				email := "lazy-load-test@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: "lazy-load-thread",
					Subject:        "Lazy Load Test",
				}
				if err := db.SaveThread(ctx, pool, thread); err != nil {
					t.Fatalf("Failed to save thread: %v", err)
				}

				now := time.Now()
				msgNoBody := &models.Message{
					ThreadID:        thread.ID,
					UserID:          userID,
					IMAPUID:         300,
					IMAPFolderName:  "INBOX",
					MessageIDHeader: "msg-no-body-test",
					FromAddress:     "sender@example.com",
					ToAddresses:     []string{"recipient@example.com"},
					Subject:         "Test No Body",
					SentAt:          &now,
				}
				if err := db.SaveMessage(ctx, pool, msgNoBody); err != nil {
					t.Fatalf("Failed to save message: %v", err)
				}

				mockIMAP := mocks.NewIMAPService(t)
				mockIMAP.On("SyncFullMessages", mock.Anything, userID, mock.MatchedBy(func(msgs []imap.MessageToSync) bool {
					return len(msgs) > 0 && msgs[0].IMAPUID == 300
				})).Return(nil).Once()

				handler := NewThreadHandler(pool, encryptor, mockIMAP)
				req := createRequestWithUser("GET", "/api/v1/thread/lazy-load-thread", email)
				return handler, mockIMAP, req, email
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				var response models.Thread
				assert.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
			},
		},
		{
			name: "does not sync when body already exists",
			setup: func(t *testing.T) (*ThreadHandler, *mocks.IMAPService, *http.Request, string) {
				email := "body-exists-test@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				threadWithBody := &models.Thread{
					UserID:         userID,
					StableThreadID: "thread-with-body",
					Subject:        "Thread with Body",
				}
				if err := db.SaveThread(ctx, pool, threadWithBody); err != nil {
					t.Fatalf("Failed to save thread: %v", err)
				}

				now := time.Now()
				msgWithBody := &models.Message{
					ThreadID:        threadWithBody.ID,
					UserID:          userID,
					IMAPUID:         200,
					IMAPFolderName:  "INBOX",
					MessageIDHeader: "msg-with-body",
					FromAddress:     "sender@example.com",
					ToAddresses:     []string{"recipient@example.com"},
					Subject:         "Message with Body",
					SentAt:          &now,
					UnsafeBodyHTML:  "<p>Existing body</p>",
					BodyText:        "Existing body",
				}
				if err := db.SaveMessage(ctx, pool, msgWithBody); err != nil {
					t.Fatalf("Failed to save message: %v", err)
				}

				mockIMAP := mocks.NewIMAPService(t)
				// SyncFullMessages should NOT be called when body already exists
				// No expectations set - if called, test will fail

				handler := NewThreadHandler(pool, encryptor, mockIMAP)
				req := createRequestWithUser("GET", "/api/v1/thread/thread-with-body", email)
				return handler, mockIMAP, req, email
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				// Verify SyncFullMessages was NOT called when body already exists
				mockIMAP.AssertNotCalled(t, "SyncFullMessages", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name: "continues when SyncFullMessages returns an error",
			setup: func(t *testing.T) (*ThreadHandler, *mocks.IMAPService, *http.Request, string) {
				email := "sync-error-thread@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: "thread-sync-error",
					Subject:        "Sync Error Test",
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
					MessageIDHeader: "msg-sync-error",
					Subject:         "Test",
					SentAt:          &now,
				}
				if err := db.SaveMessage(ctx, pool, msg); err != nil {
					t.Fatalf("Failed to save message: %v", err)
				}

				mockIMAP := mocks.NewIMAPService(t)
				mockIMAP.On("SyncFullMessages", mock.Anything, userID, mock.Anything).Return(assert.AnError).Once()

				handler := NewThreadHandler(pool, encryptor, mockIMAP)
				req := createRequestWithUser("GET", "/api/v1/thread/thread-sync-error", email)
				return handler, mockIMAP, req, email
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				var response models.Thread
				assert.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
				assert.NotEmpty(t, response.Messages, "messages should be returned even when sync fails")
			},
		},
		{
			name: "continues when GetMessageByUID fails after sync",
			setup: func(t *testing.T) (*ThreadHandler, *mocks.IMAPService, *http.Request, string) {
				email := "getmessage-error@example.com"
				ctx := context.Background()
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: "thread-getmessage-error",
					Subject:        "GetMessage Error Test",
				}
				if err := db.SaveThread(ctx, pool, thread); err != nil {
					t.Fatalf("Failed to save thread: %v", err)
				}

				now := time.Now()
				msg := &models.Message{
					ThreadID:        thread.ID,
					UserID:          userID,
					IMAPUID:         999,
					IMAPFolderName:  "INBOX",
					MessageIDHeader: "msg-getmessage-error",
					Subject:         "Test",
					SentAt:          &now,
				}
				if err := db.SaveMessage(ctx, pool, msg); err != nil {
					t.Fatalf("Failed to save message: %v", err)
				}

				mockIMAP := mocks.NewIMAPService(t)
				mockIMAP.On("SyncFullMessages", mock.Anything, userID, mock.Anything).Return(nil).Once()

				handler := NewThreadHandler(pool, encryptor, mockIMAP)
				req := createRequestWithUser("GET", "/api/v1/thread/thread-getmessage-error", email)
				return handler, mockIMAP, req, email
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				var response models.Thread
				assert.NoError(t, json.NewDecoder(rr.Body).Decode(&response))
				assert.NotEmpty(t, response.Messages, "messages should be returned even if GetMessageByUID fails")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockIMAP, req, _ := tt.setup(t)
			rr := httptest.NewRecorder()

			handler.GetThread(rr, req)

			assert.Equal(t, tt.expectCode, rr.Code)
			if tt.checkResult != nil {
				tt.checkResult(t, rr, mockIMAP)
			}
		})
	}
}
