package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
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

	t.Run("handles multiple spaces between Bearer and token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer    valid_token_12345")

		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	t.Run("handles case-insensitive Bearer scheme", func(t *testing.T) {
		testCases := []string{
			"bearer valid_token_12345",
			"BEARER valid_token_12345",
			"BeArEr valid_token_12345",
			"Bearer valid_token_12345",
		}

		for _, authHeader := range testCases {
			t.Run(authHeader, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", authHeader)

				rr := httptest.NewRecorder()
				authHandler.ServeHTTP(rr, req)

				if rr.Code != http.StatusOK {
					t.Errorf("Expected status 200 for %s, got %d", authHeader, rr.Code)
				}
			})
		}
	})

	t.Run("handles token with leading/trailing whitespace", func(t *testing.T) {
		testCases := []struct {
			name  string
			token string
		}{
			{"leading space", " Bearer valid_token_12345"},
			{"trailing space", "Bearer valid_token_12345 "},
			{"both spaces", "Bearer  valid_token_12345  "},
			{"tabs", "Bearer\tvalid_token_12345\t"},
			{"newlines", "Bearer\nvalid_token_12345\n"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", tc.token)

				rr := httptest.NewRecorder()
				authHandler.ServeHTTP(rr, req)

				if rr.Code != http.StatusOK {
					t.Errorf("Expected status 200 for %s, got %d", tc.name, rr.Code)
				}
			})
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

	t.Run("extracts email from token when VMAIL_TEST_MODE=true", func(t *testing.T) {
		originalValue := os.Getenv("VMAIL_TEST_MODE")
		defer func(key, value string) {
			err := os.Setenv(key, value)
			if err != nil {
				t.Fatalf("Failed to restore %s: %v", key, err)
			}
		}("VMAIL_TEST_MODE", originalValue)

		err := os.Setenv("VMAIL_TEST_MODE", "true")
		if err != nil {
			t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
			return
		}

		email, err := ValidateToken("email:testuser@example.com")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if email != "testuser@example.com" {
			t.Errorf("Expected email 'testuser@example.com', got %s", email)
		}
	})

	t.Run("returns error for empty token", func(t *testing.T) {
		testCases := []string{"", "   ", "\t", "\n"}
		for _, token := range testCases {
			_, err := ValidateToken(token)
			if err == nil {
				t.Errorf("Expected error for empty/whitespace token: %q", token)
			}
		}
	})

	t.Run("returns error when VMAIL_TEST_MODE=true and token is email: with empty email", func(t *testing.T) {
		originalValue := os.Getenv("VMAIL_TEST_MODE")
		defer func(key, value string) {
			err := os.Setenv(key, value)
			if err != nil {
				t.Fatalf("Failed to restore %s: %v", key, err)
			}
		}("VMAIL_TEST_MODE", originalValue)

		err := os.Setenv("VMAIL_TEST_MODE", "true")
		if err != nil {
			t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
			return
		}

		testCases := []string{"email:", "email:   ", "email:\t"}
		for _, token := range testCases {
			_, err := ValidateToken(token)
			if err == nil {
				t.Errorf("Expected error for token with empty email: %q", token)
			}
		}
	})
}
