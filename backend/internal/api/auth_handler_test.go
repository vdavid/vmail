package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestAuthHandler_GetAuthStatus(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	handler := NewAuthHandler(pool)

	t.Run("returns 401 when no user email in context", func(t *testing.T) {
		VerifyAuthCheck(t, handler.GetAuthStatus, "GET", "/api/v1/auth/status")
	})

	t.Run("returns isSetupComplete false for new user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)
		ctx := context.WithValue(req.Context(), auth.UserEmailKey, "newuser@example.com")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.GetAuthStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.AuthStatusResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.False(t, response.IsSetupComplete)
	})

	t.Run("returns isSetupComplete true for user with settings", func(t *testing.T) {
		email := "setupuser@example.com"

		// We need to create settings. We can use setupTestUserAndSettings helper
		// but we need an encryptor for that. Or just do it manually since we don't need encryption here really.
		// Let's use the helper if we import getTestEncryptor from api_test_helpers.go
		// But wait, getTestEncryptor is in the same package, so it's available.
		encryptor := getTestEncryptor(t)
		setupTestUserAndSettings(t, pool, encryptor, email)

		req := createRequestWithUser("GET", "/api/v1/auth/status", email)
		rr := httptest.NewRecorder()
		handler.GetAuthStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.AuthStatusResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.True(t, response.IsSetupComplete)
	})

	t.Run("returns 500 when GetOrCreateUser returns an error", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)
		reqCtx := context.WithValue(canceledCtx, auth.UserEmailKey, "test@example.com")
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetAuthStatus(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("returns 500 when UserSettingsExist returns an error", func(t *testing.T) {
		email := "erroruser@example.com"
		// Create user first with valid context
		_, err := db.GetOrCreateUser(context.Background(), pool, email)
		assert.NoError(t, err)

		// Use a context with a deadline that's already passed to cause UserSettingsExist to fail
		deadlineCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		reqCtx := context.WithValue(deadlineCtx, auth.UserEmailKey, email)

		req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		handler.GetAuthStatus(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
