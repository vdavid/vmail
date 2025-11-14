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

// RequireAuth middleware checks for a valid bearer token in the Authorization header.
// It extracts the token, validates it, and stores the user's email in the request context
// for use by downstream handlers. Returns 401 Unauthorized if authentication fails.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			log.Println("Auth: No Authorization header present")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse Authorization header: "Bearer <token>" (RFC 7235)
		// Use strings.Fields to handle multiple spaces and trim whitespace
		// Bearer scheme is case-insensitive per RFC 7235
		fields := strings.Fields(authHeader)
		if len(fields) < 2 {
			log.Println("Auth: Invalid Authorization header format")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if the scheme is "Bearer" (case-insensitive)
		if !strings.EqualFold(fields[0], "Bearer") {
			log.Println("Auth: Invalid Authorization header format")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Join remaining fields to handle tokens that may contain spaces
		// (though typically tokens don't, this is more robust)
		token := strings.TrimSpace(strings.Join(fields[1:], " "))
		if token == "" {
			log.Println("Auth: Empty token after Bearer")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userEmail, err := ValidateToken(token)
		if err != nil {
			log.Printf("Auth: Token validation failed: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserEmailKey, userEmail)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
