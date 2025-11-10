package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
)

// SearchHandler handles search-related API requests.
type SearchHandler struct {
	pool        *pgxpool.Pool
	encryptor   *crypto.Encryptor
	imapService imap.IMAPService
}

// NewSearchHandler creates a new SearchHandler instance.
func NewSearchHandler(pool *pgxpool.Pool, encryptor *crypto.Encryptor, imapService imap.IMAPService) *SearchHandler {
	return &SearchHandler{
		pool:        pool,
		encryptor:   encryptor,
		imapService: imapService,
	}
}

// Search handles search requests.
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := h.getUserIDFromContext(ctx, w)
	if !ok {
		return
	}

	// Get query from query param
	query := r.URL.Query().Get("q")
	// Empty query means return all emails

	// Get pagination params
	page, limitFromQuery := parsePaginationParams(r, 100)
	limit := h.getPaginationLimit(ctx, userID, limitFromQuery)

	// Call IMAP service search
	threads, totalCount, err := h.imapService.Search(ctx, userID, query, page, limit)
	if err != nil {
		// Check if it's a query parsing error (should return 400)
		if strings.Contains(err.Error(), "invalid search query") {
			log.Printf("SearchHandler: Invalid query: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("SearchHandler: Failed to search: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Build and send the response
	response := buildPaginationResponse(threads, totalCount, page, limit)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("SearchHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *SearchHandler) getUserIDFromContext(ctx context.Context, w http.ResponseWriter) (string, bool) {
	email, ok := auth.GetUserEmailFromContext(ctx)
	if !ok {
		log.Println("SearchHandler: No user email in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}

	userID, err := db.GetOrCreateUser(ctx, h.pool, email)
	if err != nil {
		log.Printf("SearchHandler: Failed to get/create user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return "", false
	}

	return userID, true
}

func (h *SearchHandler) getPaginationLimit(ctx context.Context, userID string, limitFromQuery int) int {
	if limitFromQuery > 0 {
		return limitFromQuery
	}

	settings, err := db.GetUserSettings(ctx, h.pool, userID)
	if err == nil {
		return settings.PaginationThreadsPerPage
	}

	return 100
}
