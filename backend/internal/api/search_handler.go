package api

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
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
	limit := h.getPaginationLimit(ctx, userID, limitFromQuery)

	// Call IMAP service search
	threads, totalCount, err := h.imapService.Search(ctx, userID, query, page, limit)
	if err != nil {
		// FIXME-SMELL: Error handling uses strings.Contains which is fragile.
		// If the error message changes or is wrapped differently, this check will fail.
		// Consider using error wrapping with a sentinel error type in the IMAP package
		// (e.g., ErrInvalidSearchQuery) and checking with errors.Is() instead.
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
	response := BuildPaginationResponse(threads, totalCount, page, limit)
	if !WriteJSONResponse(w, response) {
		return
	}
}

// getPaginationLimit gets the pagination limit, using user settings if available.
// If limitFromQuery is provided (> 0), it takes precedence.
// Otherwise, it uses the user's setting from the database, or defaults to 100.
// FIXME-SIMPLIFY: This function is duplicated in threads_handler.go.
// Consider extracting it to helpers.go as a shared function (e.g., GetPaginationLimit)
// that takes the pool and context as parameters.
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
