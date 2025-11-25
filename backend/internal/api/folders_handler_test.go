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
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
	"github.com/vdavid/vmail/backend/internal/testutil/mocks"
)

func TestFoldersHandler_GetFolders(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()
	encryptor := getTestEncryptor(t)

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		mockPool := mocks.NewIMAPPool(t)
		handler := NewFoldersHandler(pool, encryptor, mockPool)
		VerifyAuthCheck(t, handler.GetFolders, "GET", "/api/v1/folders")
	})

	t.Run("returns 404 when user settings not found", func(t *testing.T) {
		email := "newuser@example.com"
		mockPool := mocks.NewIMAPPool(t)
		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := createRequestWithUser("GET", "/api/v1/folders", email)
		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns 500 when GetOrCreateUser returns an error", func(t *testing.T) {
		email := "dberror@example.com"
		// Use a canceled context to simulate database connection failure
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		mockPool := mocks.NewIMAPPool(t)
		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := httptest.NewRequest("GET", "/api/v1/folders", nil)
		reqCtx := context.WithValue(canceledCtx, auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("returns 500 when decrypting IMAP password fails", func(t *testing.T) {
		email := "decrypt-error@example.com"
		ctx := context.Background()
		userID, err := db.GetOrCreateUser(ctx, pool, email)
		assert.NoError(t, err)

		// Save settings with corrupted password (but valid SMTP password to satisfy NOT NULL constraint)
		corruptedPassword := []byte("not-valid-encrypted-data")
		encryptedSMTPPassword, _ := encryptor.Encrypt("smtp_pass")
		settings := &models.UserSettings{
			UserID:                userID,
			IMAPServerHostname:    "imap.test.com",
			IMAPUsername:          "user",
			EncryptedIMAPPassword: corruptedPassword,
			SMTPServerHostname:    "smtp.test.com",
			SMTPUsername:          "user",
			EncryptedSMTPPassword: encryptedSMTPPassword,
		}
		err = db.SaveUserSettings(ctx, pool, settings)
		assert.NoError(t, err)

		mockPool := mocks.NewIMAPPool(t)
		handler := NewFoldersHandler(pool, encryptor, mockPool)

		req := createRequestWithUser("GET", "/api/v1/folders", email)
		rr := httptest.NewRecorder()
		handler.GetFolders(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestFoldersHandler_GetFolders_Scenarios(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()
	encryptor := getTestEncryptor(t)

	tests := []struct {
		name           string
		setupMock      func(*mocks.IMAPPool, *mocks.IMAPClient)
		expectedStatus int
		expectedBody   string // substring match
		setupSettings  bool   // default true
	}{
		{
			name: "success",
			setupMock: func(mp *mocks.IMAPPool, mc *mocks.IMAPClient) {
				folders := []*models.Folder{
					{Name: "INBOX", Role: "inbox"},
					{Name: "Sent", Role: "sent"},
				}
				mc.On("ListFolders").Return(folders, nil)

				mp.On("WithClient", mock.Anything, "imap.test.com", "user", mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(4).(func(imap.IMAPClient) error)
						fn(mc)
					}).
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "INBOX",
		},
		{
			name: "IMAP connection error",
			setupMock: func(mp *mocks.IMAPPool, mc *mocks.IMAPClient) {
				mp.On("WithClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("connection failed"))
				// RemoveClient is NOT called for non-retryable connection errors
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "ListFolders error",
			setupMock: func(mp *mocks.IMAPPool, mc *mocks.IMAPClient) {
				mc.On("ListFolders").Return(nil, fmt.Errorf("list failed"))

				mp.On("WithClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(4).(func(imap.IMAPClient) error)
						_ = fn(mc) // error returned by fn is returned by WithClient
					}).
					Return(fmt.Errorf("list failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "SPECIAL-USE error",
			setupMock: func(mp *mocks.IMAPPool, mc *mocks.IMAPClient) {
				mc.On("ListFolders").Return(nil, fmt.Errorf("IMAP server does not support SPECIAL-USE"))

				mp.On("WithClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(args mock.Arguments) {
						fn := args.Get(4).(func(imap.IMAPClient) error)
						_ = fn(mc)
					}).
					Return(fmt.Errorf("IMAP server does not support SPECIAL-USE"))
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "SPECIAL-USE",
		},
		{
			name: "timeout error",
			setupMock: func(mp *mocks.IMAPPool, mc *mocks.IMAPClient) {
				mp.On("WithClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(fmt.Errorf("dial tcp: i/o timeout"))
				// RemoveClient is NOT called for timeout errors (only for broken pipe/reset/EOF)
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := fmt.Sprintf("test-%s@example.com", strings.ReplaceAll(tt.name, " ", "-"))
			// Default to setting up user settings unless explicitly disabled
			if tt.setupSettings || !strings.Contains(tt.name, "decrypt") {
				setupTestUserAndSettings(t, pool, encryptor, email)
			}

			mockPool := mocks.NewIMAPPool(t)
			mockClient := mocks.NewIMAPClient(t)
			if tt.setupMock != nil {
				tt.setupMock(mockPool, mockClient)
			}

			handler := NewFoldersHandler(pool, encryptor, mockPool)
			req := createRequestWithUser("GET", "/api/v1/folders", email)
			rr := httptest.NewRecorder()
			handler.GetFolders(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestSortFoldersByRole(t *testing.T) {
	tests := []struct {
		name     string
		folders  []*models.Folder
		expected []string
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
		// ... add more cases if needed, or keep it simple
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			folders := make([]*models.Folder, len(tt.folders))
			copy(folders, tt.folders)

			sortFoldersByRole(folders)

			var names []string
			for _, f := range folders {
				names = append(names, f.Name)
			}
			assert.Equal(t, tt.expected, names)
		})
	}
}
