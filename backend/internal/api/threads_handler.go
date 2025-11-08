package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
)

type ThreadsHandler struct {
	pool        *pgxpool.Pool
	encryptor   *crypto.Encryptor
	imapService imap.IMAPService
}

func NewThreadsHandler(pool *pgxpool.Pool, encryptor *crypto.Encryptor, imapService imap.IMAPService) *ThreadsHandler {
	return &ThreadsHandler{
		pool:        pool,
		encryptor:   encryptor,
		imapService: imapService,
	}
}

func (h *ThreadsHandler) GetThreads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := h.getUserIDFromContext(ctx, w)
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
	page := 1
	limit := 100 // Default fallback
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if parsed, err := strconv.Atoi(pageStr); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	} else {
		// If no limit provided, use user's setting as default
		settings, err := db.GetUserSettings(ctx, h.pool, userID)
		if err == nil {
			limit = settings.PaginationThreadsPerPage
		}
		// If settings not found, use default 100 (already set above)
	}
	offset := (page - 1) * limit

	// Check if we should sync
	shouldSync, err := h.imapService.ShouldSyncFolder(ctx, userID, folder)
	if err != nil {
		log.Printf("ThreadsHandler: Failed to check cache: %v", err)
		// Continue anyway - try to sync
		shouldSync = true
	}

	if shouldSync {
		log.Printf("ThreadsHandler: Syncing folder %s for user %s", folder, userID)
		if err := h.imapService.SyncThreadsForFolder(ctx, userID, folder); err != nil {
			log.Printf("ThreadsHandler: Failed to sync folder: %v", err)
			// Continue anyway - return cached data if available
		}
	}

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

	// Build pagination response
	response := struct {
		Threads    []*models.Thread `json:"threads"`
		Pagination struct {
			TotalCount int `json:"total_count"`
			Page       int `json:"page"`
			PerPage    int `json:"per_page"`
		} `json:"pagination"`
	}{
		Threads: threads,
	}
	response.Pagination.TotalCount = totalCount
	response.Pagination.Page = page
	response.Pagination.PerPage = limit

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("ThreadsHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *ThreadsHandler) getUserIDFromContext(ctx context.Context, w http.ResponseWriter) (string, bool) {
	email, ok := auth.GetUserEmailFromContext(ctx)
	if !ok {
		log.Println("ThreadsHandler: No user email in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}

	userID, err := db.GetOrCreateUser(ctx, h.pool, email)
	if err != nil {
		log.Printf("ThreadsHandler: Failed to get/create user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return "", false
	}

	return userID, true
}
