package api

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
)

// GetUserIDFromContext extracts the user's email from context, resolves/creates the DB user,
// and writes appropriate HTTP errors when it fails. Returns (userID, true) on success.
// This is a shared helper function used across multiple handlers to ensure consistent
// error handling for user authentication and user ID resolution.
func GetUserIDFromContext(ctx context.Context, w http.ResponseWriter, pool *pgxpool.Pool) (string, bool) {
	email, ok := auth.GetUserEmailFromContext(ctx)
	if !ok {
		log.Println("API: No user email in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}

	userID, err := db.GetOrCreateUser(ctx, pool, email)
	if err != nil {
		log.Printf("API: Failed to get/create user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return "", false
	}

	return userID, true
}
