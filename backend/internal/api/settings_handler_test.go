package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestSettingsHandler_GetSettings(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	handler := NewSettingsHandler(pool, encryptor)

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		VerifyAuthCheck(t, handler.GetSettings, "GET", "/api/v1/settings")
	})

	t.Run("returns 404 for user without settings", func(t *testing.T) {
		email := "new-user@example.com"
		req := createRequestWithUser("GET", "/api/v1/settings", email)
		rr := httptest.NewRecorder()
		handler.GetSettings(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("returns settings for user with settings", func(t *testing.T) {
		email := "setupuser@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		req := createRequestWithUser("GET", "/api/v1/settings", email)
		rr := httptest.NewRecorder()
		handler.GetSettings(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.UserSettingsResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Equal(t, "imap.test.com", response.IMAPServerHostname)
		assert.Equal(t, 20, response.UndoSendDelaySeconds)
		assert.True(t, response.IMAPPasswordSet)
		assert.True(t, response.SMTPPasswordSet)
	})

	t.Run("returns 500 when GetUserSettings returns non-NotFound error", func(t *testing.T) {
		email := "dberror-get@example.com"
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		req := httptest.NewRequest("GET", "/api/v1/settings", nil)
		reqCtx := context.WithValue(canceledCtx, auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetSettings(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestSettingsHandler_PostSettings(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	handler := NewSettingsHandler(pool, encryptor)

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		VerifyAuthCheck(t, handler.PostSettings, "POST", "/api/v1/settings")
	})

	tests := []struct {
		name           string
		reqBody        models.UserSettingsRequest
		email          string
		setupInitial   bool
		expectedStatus int
		checkResult    func(*testing.T, string)
	}{
		{
			name:  "saves new settings successfully",
			email: "new-user@example.com",
			reqBody: models.UserSettingsRequest{
				UndoSendDelaySeconds:     25,
				PaginationThreadsPerPage: 75,
				IMAPServerHostname:       "imap.new.com",
				IMAPUsername:             "new-user",
				IMAPPassword:             "imap_password_123",
				SMTPServerHostname:       "smtp.new.com",
				SMTPUsername:             "new-user",
				SMTPPassword:             "smtp_password_456",
			},
			expectedStatus: http.StatusOK,
			checkResult: func(t *testing.T, userID string) {
				saved, err := db.GetUserSettings(context.Background(), pool, userID)
				assert.NoError(t, err)
				assert.Equal(t, "imap.new.com", saved.IMAPServerHostname)

				pass, _ := encryptor.Decrypt(saved.EncryptedIMAPPassword)
				assert.Equal(t, "imap_password_123", pass)
			},
		},
		{
			name:         "updates existing settings",
			email:        "updateuser@example.com",
			setupInitial: true,
			reqBody: models.UserSettingsRequest{
				UndoSendDelaySeconds:     40,
				PaginationThreadsPerPage: 200,
				IMAPServerHostname:       "new.imap.com",
				IMAPUsername:             "new_user",
				IMAPPassword:             "new_imap_password",
				SMTPServerHostname:       "new.smtp.com",
				SMTPUsername:             "new_user",
				SMTPPassword:             "new_smtp_password",
			},
			expectedStatus: http.StatusOK,
			checkResult: func(t *testing.T, userID string) {
				updated, err := db.GetUserSettings(context.Background(), pool, userID)
				assert.NoError(t, err)
				assert.Equal(t, "new.imap.com", updated.IMAPServerHostname)
			},
		},
		{
			name:           "returns 400 for invalid request body",
			email:          "user@example.com",
			reqBody:        models.UserSettingsRequest{}, // Will be overridden by raw bytes in test loop if needed, but here we just rely on marshalling failing or empty values triggering validation
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "updates settings without passwords when passwords are empty",
			email:        "updatewithoutpass@example.com",
			setupInitial: true,
			reqBody: models.UserSettingsRequest{
				UndoSendDelaySeconds:     40,
				PaginationThreadsPerPage: 200,
				IMAPServerHostname:       "new.imap.com",
				IMAPUsername:             "new_user",
				IMAPPassword:             "", // Empty
				SMTPServerHostname:       "new.smtp.com",
				SMTPUsername:             "new_user",
				SMTPPassword:             "", // Empty
			},
			expectedStatus: http.StatusOK,
			checkResult: func(t *testing.T, userID string) {
				updated, err := db.GetUserSettings(context.Background(), pool, userID)
				assert.NoError(t, err)

				// Should preserve original "imap_pass" set by setupInitial (via setupTestUserAndSettings internal logic or custom)
				// Wait, setupInitial uses `setupTestUserAndSettings` which sets "imap_pass"
				pass, _ := encryptor.Decrypt(updated.EncryptedIMAPPassword)
				assert.Equal(t, "imap_pass", pass)
			},
		},
		{
			name:  "returns 400 when passwords are empty for new user",
			email: "newuser-nopass@example.com",
			reqBody: models.UserSettingsRequest{
				IMAPServerHostname: "imap.new.com",
				IMAPUsername:       "user",
				SMTPServerHostname: "smtp.new.com",
				SMTPUsername:       "user",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "validates missing IMAP server hostname",
			email: "val-hostname@example.com",
			reqBody: models.UserSettingsRequest{
				IMAPUsername: "user", IMAPPassword: "pw", SMTPServerHostname: "h", SMTPUsername: "u", SMTPPassword: "pw",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var userID string
			if tt.setupInitial {
				userID = setupTestUserAndSettings(t, pool, encryptor, tt.email)
			}

			var body []byte
			if tt.name == "returns 400 for invalid request body" {
				body = []byte("invalid json")
			} else {
				body, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest("POST", "/api/v1/settings", bytes.NewReader(body))
			ctx := context.WithValue(req.Context(), auth.UserEmailKey, tt.email)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.PostSettings(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResult != nil {
				// Need userID if not setup initially
				if !tt.setupInitial {
					var err error
					userID, err = db.GetOrCreateUser(context.Background(), pool, tt.email)
					assert.NoError(t, err)
				}
				tt.checkResult(t, userID)
			}

			if tt.expectedStatus == http.StatusBadRequest && tt.name != "returns 400 for invalid request body" {
				// Check for validation messages if relevant
				if strings.Contains(tt.name, "hostname") {
					assert.Contains(t, rr.Body.String(), "hostname is required")
				}
			}
		})
	}
}

func TestSettingsHandler_WriteResponseErrors(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptor(t)
	handler := NewSettingsHandler(pool, encryptor)

	t.Run("handles write failure gracefully in GetSettings", func(t *testing.T) {
		email := "write-error-get@example.com"
		setupTestUserAndSettings(t, pool, encryptor, email)

		req := createRequestWithUser("GET", "/api/v1/settings", email)
		rr := httptest.NewRecorder()
		failingWriter := &FailingResponseWriter{
			ResponseWriter:  rr,
			WriteShouldFail: true,
		}

		handler.GetSettings(failingWriter, req)
		// Check it didn't panic and set status (though write failed so body is empty)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
