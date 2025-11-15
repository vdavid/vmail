package api

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/imap"
)

// SearchHandler handles search-related API requests.
type SearchHandler struct {
	pool        *pgxpool.Pool
	encryptor   *crypto.Encryptor // Not used directly, but required by imapService
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

	userID, ok := GetUserIDFromContext(ctx, w, h.pool)
	if !ok {
		return
	}

	// Get query from query param
	query := r.URL.Query().Get("q")
	// Empty query means return all emails

	// Get pagination params
	page, limitFromQuery := ParsePaginationParams(r, 100)
	limit := GetPaginationLimit(ctx, h.pool, userID, limitFromQuery)

	// Call IMAP service search
	threads, totalCount, err := h.imapService.Search(ctx, userID, query, page, limit)
	if err != nil {
		// Treat client cancellations as non-errors
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}

		// Check if it's a query parsing error (should return 400)
		if errors.Is(err, imap.ErrInvalidSearchQuery) {
			log.Printf("SearchHandler: Invalid query: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("SearchHandler: Failed to search: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Build and send the response
	response := BuildPaginationResponse(threads, totalCount, page, limit)
	if !WriteJSONResponse(w, response) {
		return
	}
}
