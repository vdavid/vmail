package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

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

// parsePaginationParams parses page and limit from query parameters.
func parsePaginationParams(r *http.Request, defaultLimit int) (page, limit int) {
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

// buildPaginationResponse builds the pagination response structure.
func buildPaginationResponse(threads []*models.Thread, totalCount, page, limit int) interface{} {
	return struct {
		Threads    []*models.Thread `json:"threads"`
		Pagination struct {
			TotalCount int `json:"total_count"`
			Page       int `json:"page"`
			PerPage    int `json:"per_page"`
		} `json:"pagination"`
	}{
		Threads: threads,
		Pagination: struct {
			TotalCount int `json:"total_count"`
			Page       int `json:"page"`
			PerPage    int `json:"per_page"`
		}{
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
	page, limitFromQuery := parsePaginationParams(r, 100)
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
	response := buildPaginationResponse(threads, totalCount, page, limit)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ThreadsHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
