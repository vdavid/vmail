package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
)

func setupTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	connString := "postgres://vmail:vmail@localhost:5432/vmail_test?sslmode=disable"

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Skipf("Skipping test: could not connect to test database: %v", err)
		return nil
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Skipping test: could not ping test database: %v", err)
		return nil
	}

	return pool
}

func cleanupTestPool(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()
	_, _ = pool.Exec(ctx, "TRUNCATE users CASCADE")
}

func TestAuthHandler_GetAuthStatus(t *testing.T) {
	pool := setupTestPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()
	defer cleanupTestPool(t, pool)

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

		if !response.IsAuthenticated {
			t.Error("Expected isAuthenticated to be true")
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
			ArchiveFolderName:        "Archive",
			SentFolderName:           "Sent",
			DraftsFolderName:         "Drafts",
			TrashFolderName:          "Trash",
			SpamFolderName:           "Spam",
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

		if !response.IsAuthenticated {
			t.Error("Expected isAuthenticated to be true")
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
}
