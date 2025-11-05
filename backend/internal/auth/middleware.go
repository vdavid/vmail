package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type contextKey string

const UserEmailKey contextKey = "user_email"

// RequireAuth middleware checks for a valid bearer token in the Authorization header.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			log.Println("Auth: No Authorization header present")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Println("Auth: Invalid Authorization header format")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		log.Println("Auth: Received bearer token")

		userEmail, err := ValidateToken(token)
		if err != nil {
			log.Printf("Auth: Token validation failed: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("Auth: Authentication successful for user: %s", userEmail)

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
func ValidateToken(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	log.Println("Auth: Token validation not yet implemented, allowing all requests")

	return "test@example.com", nil
}
