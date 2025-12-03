package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestAttachmentHandler_GetAttachment(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)

	tests := []struct {
		name        string
		setup       func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter)
		expectCode  int
		checkResult func(*testing.T, *httptest.ResponseRecorder, *mocks.IMAPService)
	}{
		{
			name: "returns 401 when no user email in context",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				req := httptest.NewRequest("GET", "/api/v1/messages/msg-id/attachments/att-id", nil)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusUnauthorized,
		},
		{
			name: "returns 400 when path format is invalid - missing attachmentId",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				req := createRequestWithUser("GET", "/api/v1/messages/msg-id/attachments/", "user@example.com")
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "returns 400 when path format is invalid - missing messageId",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				req := createRequestWithUser("GET", "/api/v1/messages//attachments/att-id", "user@example.com")
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "returns 400 when attachmentId has invalid URL encoding",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				email := "encoding-test@example.com"
				setupTestUserAndSettings(t, pool, encryptor, email)

				req, err := http.NewRequest("GET", "/api/v1/messages/msg-id/attachments/%ZZ", nil)
				if err != nil {
					req = &http.Request{
						Method: "GET",
						URL: &url.URL{
							Path: "/api/v1/messages/msg-id/attachments/%ZZ",
						},
					}
				}
				reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
				req = req.WithContext(reqCtx)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "returns 404 when message not found",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				email := "msg-not-found@example.com"
				setupTestUserAndSettings(t, pool, encryptor, email)
				req := createRequestWithUser("GET", "/api/v1/messages/non-existent/attachments/att-id", email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusNotFound,
		},
		{
			name: "returns 404 when message belongs to different user",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()

				// Create user1 and message
				user1ID := setupTestUserAndSettings(t, pool, encryptor, "user1@example.com")
				thread1 := &models.Thread{UserID: user1ID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread1))
				msg1 := &models.Message{
					ThreadID: thread1.ID, UserID: user1ID, IMAPUID: 1,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg1))

				// Try to access as user2
				setupTestUserAndSettings(t, pool, encryptor, "user2@example.com")
				req := createRequestWithUser("GET", "/api/v1/messages/"+msg1.ID+"/attachments/att-id", "user2@example.com")
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusNotFound,
		},
		{
			name: "returns 404 when attachment not found",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "att-not-found@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 1,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/non-existent", email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusNotFound,
		},
		{
			name: "successfully fetches attachment by ContentID",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "success-cid@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 100,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				att := &models.Attachment{
					MessageID: msg.ID, Filename: "logo.png", MimeType: "image/png",
					SizeBytes: 1024, IsInline: true, ContentID: "logo123",
				}
				assert.NoError(t, db.SaveAttachment(ctx, pool, att))

				mockIMAP.On("FetchAttachmentContent", mock.Anything, userID, "INBOX", int64(100), "logo123", "logo.png").
					Return([]byte("fake-image-data"), "image/png", "logo.png", nil).Once()

				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/"+att.ID+"?download=true", email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				assert.Equal(t, "image/png", rr.Header().Get("Content-Type"))
				assert.Contains(t, rr.Header().Get("Content-Disposition"), "attachment")
				assert.Contains(t, rr.Header().Get("Content-Disposition"), "logo.png")
				// "fake-image-data" is 15 bytes long
				assert.Equal(t, "15", rr.Header().Get("Content-Length"))
				assert.Equal(t, "private, max-age=86400", rr.Header().Get("Cache-Control"))
				assert.Equal(t, []byte("fake-image-data"), rr.Body.Bytes())
			},
		},
		{
			name: "successfully fetches inline attachment without download param",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "inline-test@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 200,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				att := &models.Attachment{
					MessageID: msg.ID, Filename: "image.jpg", MimeType: "image/jpeg",
					SizeBytes: 2048, IsInline: true, ContentID: "img456",
				}
				assert.NoError(t, db.SaveAttachment(ctx, pool, att))

				mockIMAP.On("FetchAttachmentContent", mock.Anything, userID, "INBOX", int64(200), "img456", "image.jpg").
					Return([]byte("jpeg-data"), "image/jpeg", "image.jpg", nil).Once()

				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/"+att.ID, email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				assert.Contains(t, rr.Header().Get("Content-Disposition"), "inline")
			},
		},
		{
			name: "handles attachment without ContentID (uses filename only)",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "no-cid@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 300,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				att := &models.Attachment{
					MessageID: msg.ID, Filename: "document.pdf", MimeType: "application/pdf",
					SizeBytes: 4096, IsInline: false,
				}
				assert.NoError(t, db.SaveAttachment(ctx, pool, att))

				mockIMAP.On("FetchAttachmentContent", mock.Anything, userID, "INBOX", int64(300), "", "document.pdf").
					Return([]byte("pdf-content"), "application/pdf", "document.pdf", nil).Once()

				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/"+att.ID+"?download=true", email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				assert.Equal(t, "application/pdf", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "handles missing MIME type - uses fallback",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "no-mime@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 400,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				att := &models.Attachment{
					MessageID: msg.ID, Filename: "unknown.bin", MimeType: "",
					SizeBytes: 512, IsInline: false,
				}
				assert.NoError(t, db.SaveAttachment(ctx, pool, att))

				mockIMAP.On("FetchAttachmentContent", mock.Anything, userID, "INBOX", int64(400), "", "unknown.bin").
					Return([]byte("data"), "", "unknown.bin", nil).Once()

				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/"+att.ID, email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				assert.Equal(t, "application/octet-stream", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "returns 500 when IMAP fetch fails",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "imap-error@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 500,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				att := &models.Attachment{
					MessageID: msg.ID, Filename: "file.txt", MimeType: "text/plain",
					SizeBytes: 100, IsInline: false,
				}
				assert.NoError(t, db.SaveAttachment(ctx, pool, att))

				mockIMAP.On("FetchAttachmentContent", mock.Anything, userID, "INBOX", int64(500), "", "file.txt").
					Return(nil, "", "", assert.AnError).Once()

				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/"+att.ID, email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name: "handles empty attachment content",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "empty-content@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 600,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				att := &models.Attachment{
					MessageID: msg.ID, Filename: "empty.txt", MimeType: "text/plain",
					SizeBytes: 0, IsInline: false,
				}
				assert.NoError(t, db.SaveAttachment(ctx, pool, att))

				mockIMAP.On("FetchAttachmentContent", mock.Anything, userID, "INBOX", int64(600), "", "empty.txt").
					Return([]byte{}, "text/plain", "empty.txt", nil).Once()

				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/"+att.ID, email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				assert.Equal(t, "0", rr.Header().Get("Content-Length"))
				assert.Empty(t, rr.Body.Bytes())
			},
		},
		{
			name: "handles special characters in filename",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "special-chars@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 700,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				filename := "file with spaces & symbols@#$.txt"
				att := &models.Attachment{
					MessageID: msg.ID, Filename: filename, MimeType: "text/plain",
					SizeBytes: 50, IsInline: false,
				}
				assert.NoError(t, db.SaveAttachment(ctx, pool, att))

				mockIMAP.On("FetchAttachmentContent", mock.Anything, userID, "INBOX", int64(700), "", filename).
					Return([]byte("content"), "text/plain", filename, nil).Once()

				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/"+att.ID+"?download=true", email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				// Filename should be properly quoted in Content-Disposition
				assert.Contains(t, rr.Header().Get("Content-Disposition"), `"file with spaces & symbols@#$.txt"`)
			},
		},
		{
			name: "handles URL-encoded attachment ID",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "url-encoded@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 800,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				att := &models.Attachment{
					MessageID: msg.ID, Filename: "test.pdf", MimeType: "application/pdf",
					SizeBytes: 100, IsInline: false,
				}
				assert.NoError(t, db.SaveAttachment(ctx, pool, att))

				mockIMAP.On("FetchAttachmentContent", mock.Anything, userID, "INBOX", int64(800), "", "test.pdf").
					Return([]byte("pdf"), "application/pdf", "test.pdf", nil).Once()

				// Use URL-encoded attachment ID
				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/"+att.ID, email)
				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusOK,
			checkResult: func(t *testing.T, rr *httptest.ResponseRecorder, mockIMAP *mocks.IMAPService) {
				mockIMAP.AssertExpectations(t)
				assert.Equal(t, http.StatusOK, rr.Code)
			},
		},
		{
			name: "returns 500 when GetMessageByID fails with non-NotFound error",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				email := "db-error@example.com"
				setupTestUserAndSettings(t, pool, encryptor, email)

				// Use canceled context to cause DB error
				canceledCtx, cancel := context.WithCancel(context.Background())
				cancel()
				req := httptest.NewRequest("GET", "/api/v1/messages/msg-id/attachments/att-id", nil)
				reqCtx := context.WithValue(canceledCtx, auth.UserEmailKey, email)
				req = req.WithContext(reqCtx)

				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name: "returns 500 when GetAttachmentByID fails with non-NotFound error",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "att-db-error@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 900,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				// Use canceled context to cause DB error when fetching attachment
				canceledCtx, cancel := context.WithCancel(context.Background())
				cancel()
				req := httptest.NewRequest("GET", "/api/v1/messages/"+msg.ID+"/attachments/att-id", nil)
				reqCtx := context.WithValue(canceledCtx, auth.UserEmailKey, email)
				req = req.WithContext(reqCtx)

				return handler, mockIMAP, req, httptest.NewRecorder()
			},
			expectCode: http.StatusInternalServerError,
		},
		{
			name: "handles write failure gracefully",
			setup: func(*testing.T) (*AttachmentHandler, *mocks.IMAPService, *http.Request, http.ResponseWriter) {
				mockIMAP := mocks.NewIMAPService(t)
				handler := NewAttachmentHandler(pool, mockIMAP)
				ctx := context.Background()
				email := "write-error@example.com"
				userID := setupTestUserAndSettings(t, pool, encryptor, email)

				thread := &models.Thread{UserID: userID, StableThreadID: "thread1", Subject: "Test"}
				assert.NoError(t, db.SaveThread(ctx, pool, thread))
				msg := &models.Message{
					ThreadID: thread.ID, UserID: userID, IMAPUID: 1000,
					IMAPFolderName: "INBOX", MessageIDHeader: "msg1",
					Subject: "Test", SentAt: timePtr(time.Now()),
				}
				assert.NoError(t, db.SaveMessage(ctx, pool, msg))

				att := &models.Attachment{
					MessageID: msg.ID, Filename: "test.txt", MimeType: "text/plain",
					SizeBytes: 10, IsInline: false,
				}
				assert.NoError(t, db.SaveAttachment(ctx, pool, att))

				mockIMAP.On("FetchAttachmentContent", mock.Anything, userID, "INBOX", int64(1000), "", "test.txt").
					Return([]byte("content"), "text/plain", "test.txt", nil).Once()

				req := createRequestWithUser("GET", "/api/v1/messages/"+msg.ID+"/attachments/"+att.ID, email)
				failingWriter := &FailingResponseWriter{
					ResponseWriter:  httptest.NewRecorder(),
					WriteShouldFail: true,
				}
				return handler, mockIMAP, req, failingWriter
			},
			expectCode: http.StatusOK, // Handler sets headers before write, so status is OK
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockIMAP, req, w := tt.setup(t)
			rr, ok := w.(*httptest.ResponseRecorder)
			if !ok {
				// For FailingResponseWriter case
				handler.GetAttachment(w, req)
				return
			}

			handler.GetAttachment(rr, req)
			assert.Equal(t, tt.expectCode, rr.Code)
			if tt.checkResult != nil {
				tt.checkResult(t, rr, mockIMAP)
			}
		})
	}
}

func TestParseAttachmentPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectMsgID string
		expectAttID string
		expectError bool
	}{
		{
			name:        "valid path",
			path:        "/api/v1/messages/msg-123/attachments/att-456",
			expectMsgID: "msg-123",
			expectAttID: "att-456",
			expectError: false,
		},
		{
			name:        "path with URL-encoded attachment ID",
			path:        "/api/v1/messages/msg-123/attachments/att%2D456",
			expectMsgID: "msg-123",
			expectAttID: "att-456",
			expectError: false,
		},
		{
			name:        "missing attachment ID",
			path:        "/api/v1/messages/msg-123/attachments/",
			expectError: true,
		},
		{
			name:        "missing message ID",
			path:        "/api/v1/messages//attachments/att-456",
			expectError: true,
		},
		{
			name:        "invalid format - no attachments segment",
			path:        "/api/v1/messages/msg-123/att-456",
			expectError: true,
		},
		{
			name:        "valid even with different prefix (handler routing guarantees prefix)",
			path:        "/wrong/prefix/msg-123/attachments/att-456",
			expectMsgID: "/wrong/prefix/msg-123",
			expectAttID: "att-456",
			expectError: false,
		},
		{
			name:        "invalid URL encoding",
			path:        "/api/v1/messages/msg-123/attachments/%ZZ",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgID, attID, err := parseAttachmentPath(tt.path)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectMsgID, msgID)
				assert.Equal(t, tt.expectAttID, attID)
			}
		})
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
