package api

import (
	"bytes"
	"context"
	"encoding/json"
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

// WriteJSONResponse writes a JSON response using a buffered approach to prevent partial writes.
// If encoding fails, it writes an error response and returns false. Otherwise returns true.
// This ensures atomic responses and consistent error handling across all handlers.
func WriteJSONResponse(w http.ResponseWriter, data interface{}) bool {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		log.Printf("API: Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return false
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Printf("API: Failed to write JSON response: %v", err)
	}
	return true
}

// GetPaginationLimit gets the pagination limit, using user settings if available.
// If limitFromQuery is provided (> 0), it takes precedence.
// Otherwise, it uses the user's setting from the database, or defaults to 100.
// This is a shared helper function used by multiple handlers for consistent pagination limit handling.
func GetPaginationLimit(ctx context.Context, pool *pgxpool.Pool, userID string, limitFromQuery int) int {
	if limitFromQuery > 0 {
		return limitFromQuery
	}

	settings, err := db.GetUserSettings(ctx, pool, userID)
	if err == nil {
		return settings.PaginationThreadsPerPage
	}

	return 100
}
