package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestAuthHandler_GetAuthStatus(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	handler := NewAuthHandler(pool)

	t.Run("returns isSetupComplete false for new user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)

		ctx := context.WithValue(req.Context(), auth.UserEmailKey, "newuser@example.com")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.GetAuthStatus(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response models.AuthStatusResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.IsSetupComplete {
			t.Error("Expected isSetupComplete to be false for new user")
		}
	})

	t.Run("returns isSetupComplete true for user with settings", func(t *testing.T) {
		email := "setupuser@example.com"

		ctx := context.Background()
		userID, err := db.GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		settings := &models.UserSettings{
			UserID:                   userID,
			UndoSendDelaySeconds:     20,
			PaginationThreadsPerPage: 100,
			IMAPServerHostname:       "imap.example.com",
			IMAPUsername:             "user",
			EncryptedIMAPPassword:    []byte("encrypted"),
			SMTPServerHostname:       "smtp.example.com",
			SMTPUsername:             "user",
			EncryptedSMTPPassword:    []byte("encrypted"),
		}
		if err := db.SaveUserSettings(ctx, pool, settings); err != nil {
			t.Fatalf("Failed to save settings: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)
		reqCtx := context.WithValue(req.Context(), auth.UserEmailKey, email)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetAuthStatus(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var response models.AuthStatusResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.IsSetupComplete {
			t.Error("Expected isSetupComplete to be true for user with settings")
		}
	})

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)

		rr := httptest.NewRecorder()
		handler.GetAuthStatus(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("returns 500 when GetOrCreateUser returns an error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)

		// Use a cancelled context to simulate database connection failure
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()
		reqCtx := context.WithValue(cancelledCtx, auth.UserEmailKey, "test@example.com")
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetAuthStatus(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})

	t.Run("returns 500 when UserSettingsExist returns an error", func(t *testing.T) {
		email := "erroruser@example.com"

		// Create user first with valid context
		ctx := context.Background()
		_, err := db.GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Use a context with a deadline that's already passed to cause UserSettingsExist to fail
		// Note: GetOrCreateUser might succeed due to ON CONFLICT, but UserSettingsExist will fail
		deadlineCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		reqCtx := context.WithValue(deadlineCtx, auth.UserEmailKey, email)

		req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetAuthStatus(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", rr.Code)
		}
	})
}
