package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
)

// ThreadsHandler handles thread-list-related API requests.
type ThreadsHandler struct {
	pool        *pgxpool.Pool
	encryptor   *crypto.Encryptor
	imapService imap.IMAPService
}

// NewThreadsHandler creates a new ThreadsHandler instance.
func NewThreadsHandler(pool *pgxpool.Pool, encryptor *crypto.Encryptor, imapService imap.IMAPService) *ThreadsHandler {
	return &ThreadsHandler{
		pool:        pool,
		encryptor:   encryptor,
		imapService: imapService,
	}
}

// getPaginationLimit gets the pagination limit, using user settings if available.
func (h *ThreadsHandler) getPaginationLimit(ctx context.Context, userID string, limitFromQuery int) int {
	if limitFromQuery > 0 {
		return limitFromQuery
	}

	// If no limit provided, use the user's setting as default
	settings, err := db.GetUserSettings(ctx, h.pool, userID)
	if err == nil {
		return settings.PaginationThreadsPerPage
	}

	// If settings not found, use default 100
	return 100
}

// syncFolderIfNeeded checks if the folder needs syncing and syncs if necessary.
func (h *ThreadsHandler) syncFolderIfNeeded(ctx context.Context, userID, folder string) {
	shouldSync, err := h.imapService.ShouldSyncFolder(ctx, userID, folder)
	if err != nil {
		log.Printf("ThreadsHandler: Failed to check cache: %v", err)
		shouldSync = true // Continue anyway - try to sync
	}

	if shouldSync {
		log.Printf("ThreadsHandler: Syncing folder %s for user %s", folder, userID)
		if err := h.imapService.SyncThreadsForFolder(ctx, userID, folder); err != nil {
			log.Printf("ThreadsHandler: Failed to sync folder: %v", err)
			// Continue anyway - return cached data if available
		}
	}
}

// BuildPaginationResponse builds the pagination response structure.
// This is a shared helper function used by multiple handlers for consistent response formatting.
func BuildPaginationResponse(threads []*models.Thread, totalCount, page, limit int) *models.ThreadsResponse {
	return &models.ThreadsResponse{
		Threads: threads,
		Pagination: models.PaginationInfo{
			TotalCount: totalCount,
			Page:       page,
			PerPage:    limit,
		},
	}
}

// GetThreads returns a paginated list of email threads for a folder.
func (h *ThreadsHandler) GetThreads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx, w, h.pool)
	if !ok {
		return
	}

	// Get folder from query param
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		http.Error(w, "folder query parameter is required", http.StatusBadRequest)
		return
	}

	// Get pagination params
	page, limitFromQuery := ParsePaginationParams(r, 100)
	limit := h.getPaginationLimit(ctx, userID, limitFromQuery)
	offset := (page - 1) * limit

	// Sync folder if needed
	h.syncFolderIfNeeded(ctx, userID, folder)

	// Get threads from the database
	threads, err := db.GetThreadsForFolder(ctx, h.pool, userID, folder, limit, offset)
	if err != nil {
		log.Printf("ThreadsHandler: Failed to get threads: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	totalCount, err := db.GetThreadCountForFolder(ctx, h.pool, userID, folder)
	if err != nil {
		log.Printf("ThreadsHandler: Failed to get thread count: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Build and send the response
	// Use a buffered approach to prevent partial writes if JSON encoding fails
	response := BuildPaginationResponse(threads, totalCount, page, limit)

	// Encode to buffer first to prevent partial writes
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(response); err != nil {
		log.Printf("ThreadsHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Only write headers and body if encoding succeeded
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Printf("ThreadsHandler: Failed to write response: %v", err)
	}
}
