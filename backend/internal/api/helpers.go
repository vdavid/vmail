package api

import (
	"context"
	"log"
	"net/http"
	"strconv"

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

// ParsePaginationParams parses page and limit from query parameters.
// Returns default values (page=1, limit=defaultLimit) if parameters are missing or invalid.
// This is a shared helper function used by multiple handlers for consistent pagination parsing.
func ParsePaginationParams(r *http.Request, defaultLimit int) (page, limit int) {
	page = 1
	limit = defaultLimit

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if parsed, err := strconv.Atoi(pageStr); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	return page, limit
}
