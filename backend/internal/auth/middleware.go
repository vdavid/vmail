package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type contextKey string

// UserEmailKey is the context key used to store the authenticated user's email.
const UserEmailKey contextKey = "user_email"

// RequireAuth middleware authenticates requests based on VMAIL_AUTH_MODE:
//   - "header": Trusts Remote-User or Remote-Email headers from reverse proxy (for production).
//   - "dev" (default): Uses Bearer token auth with stub validation (returns test@example.com).
//
// The authenticated user's email is stored in the request context for downstream handlers.
// Returns 401 Unauthorized if authentication fails.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authMode := os.Getenv("VMAIL_AUTH_MODE")

		var userEmail string
		var err error

		if authMode == "header" {
			userEmail, err = authenticateFromHeaders(r)
		} else {
			// Default to "dev" mode (Bearer token with stub validation)
			userEmail, err = authenticateFromBearerToken(r)
		}

		if err != nil {
			log.Printf("Auth: Authentication failed: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserEmailKey, userEmail)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticateFromHeaders extracts user email from reverse proxy headers.
// This is used when VMAIL_AUTH_MODE=header with Authelia + Caddy/Nginx.
// The reverse proxy sets Remote-Email or Remote-User after validating the session.
func authenticateFromHeaders(r *http.Request) (string, error) {
	// Try Remote-Email first (preferred, contains actual email)
	email := r.Header.Get("Remote-Email")
	if email != "" {
		return email, nil
	}

	// Fall back to Remote-User (may be username or email depending on Authelia config)
	user := r.Header.Get("Remote-User")
	if user != "" {
		return user, nil
	}

	return "", fmt.Errorf("no Remote-Email or Remote-User header present")
}

// authenticateFromBearerToken extracts and validates a Bearer token from the Authorization header.
// This is used in dev mode for local development.
func authenticateFromBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return "", fmt.Errorf("no Authorization header present")
	}

	// Parse Authorization header: "Bearer <token>" (RFC 7235)
	// Use strings.Fields to handle multiple spaces and trim whitespace
	// Bearer scheme is case-insensitive per RFC 7235
	fields := strings.Fields(authHeader)
	if len(fields) < 2 {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	// Check if the scheme is "Bearer" (case-insensitive)
	if !strings.EqualFold(fields[0], "Bearer") {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	// Join remaining fields to handle tokens that may contain spaces
	// (though typically tokens don't, this is more robust)
	token := strings.TrimSpace(strings.Join(fields[1:], " "))
	if token == "" {
		return "", fmt.Errorf("empty token after Bearer")
	}

	return ValidateToken(token)
}

// GetUserEmailFromContext returns the user email from the context.
func GetUserEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(UserEmailKey).(string)
	return email, ok
}

// ValidateToken validates the token and returns the user's email.
// This is a stub for now.
// In test mode (VMAIL_TEST_MODE=true), if the token starts with "email:",
// it extracts the email from the token (e.g., "email:user@example.com" -> "user@example.com").
// Otherwise, it returns "test@example.com" as the default test user.
func ValidateToken(token string) (string, error) {
	if strings.TrimSpace(token) == "" || strings.TrimSpace(token) == "email:" {
		return "", fmt.Errorf("token is empty")
	}

	// In test mode, support extracting email from token format "email:user@example.com"
	if os.Getenv("VMAIL_TEST_MODE") == "true" {
		if strings.HasPrefix(token, "email:") {
			email := strings.TrimPrefix(token, "email:")
			if email != "" {
				return email, nil
			}
		}
	}

	// TODO: Implement token validation

	return "test@example.com", nil
}
