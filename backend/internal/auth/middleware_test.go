package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAuth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email, ok := GetUserEmailFromContext(r.Context())
		if !ok {
			t.Error("Expected user email in context")
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(email))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
			return
		}
	})

	authHandler := RequireAuth(handler)

	t.Run("allows request with valid Bearer token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid_token_12345")

		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	t.Run("rejects request without Authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("rejects request with invalid Authorization format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")

		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("rejects request with wrong auth scheme", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Basic abcd_abcd_abcd")

		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	t.Run("rejects empty token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer ")

		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})
}

func TestGetUserEmailFromContext(t *testing.T) {
	t.Run("returns email when present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid_token")

		handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			email, ok := GetUserEmailFromContext(r.Context())
			if !ok {
				t.Error("Expected user email to be present in context")
			}
			if email == "" {
				t.Error("Expected non-empty email")
			}
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	})

	t.Run("returns false when not present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		email, ok := GetUserEmailFromContext(req.Context())
		if ok {
			t.Error("Expected ok to be false")
		}
		if email != "" {
			t.Error("Expected empty email")
		}
	})
}

func TestValidateToken(t *testing.T) {
	t.Run("currently allows all tokens", func(t *testing.T) {
		email, err := ValidateToken("any_token")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if email == "" {
			t.Error("Expected non-empty email")
		}
	})
}
