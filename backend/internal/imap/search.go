package imap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
)

// ErrInvalidSearchQuery is returned when a search query cannot be parsed.
var ErrInvalidSearchQuery = errors.New("invalid search query")

// parseHeaderFilter processes header filters (from:, to:, subject:).
// Returns (handled, error) where handled indicates if the token matched this filter type.
func parseHeaderFilter(token, prefix, headerName string, criteria *imap.SearchCriteria) (bool, error) {
	if !strings.HasPrefix(token, prefix) {
		return false, nil
	}
	value := strings.TrimPrefix(token, prefix)
	if value == "" {
		return false, fmt.Errorf("empty %s: value", prefix[:len(prefix)-1])
	}
	criteria.Header.Add(headerName, unquote(value))
	return true, nil
}

// parseDateFilter processes date filters (after:, before:).
// Returns (handled, error) where handled indicates if the token matched this filter type.
// For before: filters, sets the time to end of day (23:59:59.999999999).
func parseDateFilter(token, prefix string, criteria *imap.SearchCriteria) (bool, error) {
	if !strings.HasPrefix(token, prefix) {
		return false, nil
	}
	value := strings.TrimPrefix(token, prefix)
	if value == "" {
		return false, fmt.Errorf("empty %s: value", prefix[:len(prefix)-1])
	}
	date, err := parseDate(value)
	if err != nil {
		return false, fmt.Errorf("invalid date format for %s: %w", prefix[:len(prefix)-1], err)
	}
	if prefix == "after:" {
		criteria.Since = date
	} else {
		// before: - set to end of day
		date = time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())
		criteria.Before = date
	}
	return true, nil
}

// parseFolderFilter processes folder: or label: filters.
// Only the first folder: or label: filter is extracted; subsequent ones are ignored.
// Returns (handled, folder, error) where handled indicates if the token matched this filter type.
func parseFolderFilter(token string, folderFound *bool) (bool, string, error) {
	if !strings.HasPrefix(token, "folder:") && !strings.HasPrefix(token, "label:") {
		return false, "", nil
	}
	if !*folderFound {
		value := strings.TrimPrefix(token, "folder:")
		value = strings.TrimPrefix(value, "label:")
		if value == "" {
			return false, "", fmt.Errorf("empty folder: value")
		}
		*folderFound = true
		return true, unquote(value), nil
	}
	return true, "", nil
}

// parseFilterToken processes a single token and updates criteria/folder accordingly.
// Returns (handled, folder, error) where handled indicates if the token was a filter.
func parseFilterToken(token string, criteria *imap.SearchCriteria, folderFound *bool) (bool, string, error) {
	// Check for filter without value
	if strings.HasSuffix(token, ":") {
		return false, "", fmt.Errorf("empty filter value: %s", token)
	}

	// Try header filters (from:, to:, subject:)
	if handled, err := parseHeaderFilter(token, "from:", "From", criteria); err != nil {
		return false, "", err
	} else if handled {
		return true, "", nil
	}

	if handled, err := parseHeaderFilter(token, "to:", "To", criteria); err != nil {
		return false, "", err
	} else if handled {
		return true, "", nil
	}

	if handled, err := parseHeaderFilter(token, "subject:", "Subject", criteria); err != nil {
		return false, "", err
	} else if handled {
		return true, "", nil
	}

	// Try date filters (after:, before:)
	if handled, err := parseDateFilter(token, "after:", criteria); err != nil {
		return false, "", err
	} else if handled {
		return true, "", nil
	}

	if handled, err := parseDateFilter(token, "before:", criteria); err != nil {
		return false, "", err
	} else if handled {
		return true, "", nil
	}

	// Try folder filter
	handled, folder, err := parseFolderFilter(token, folderFound)
	if err != nil {
		return false, "", err
	}
	if handled {
		return true, folder, nil
	}

	return false, "", nil
}

// ParseSearchQuery parses a Gmail-like search query into IMAP SearchCriteria.
// Returns the parsed criteria, extracted folder name (or empty), and error.
// Supported syntax:
//   - from:george → criteria.Header["From"] = "george"
//   - to:alice → criteria.Header["To"] = "alice"
//   - subject:meeting → criteria.Header["Subject"] = "meeting"
//   - after:2025-01-01 → criteria.Since = time.Date(...)
//   - before:2025-12-31 → criteria.Before = time.Date(...)
//   - folder:Inbox or label:Inbox → extract folder name (returned separately)
//   - Plain text → criteria.Text = []string{text}
//   - Combinations: from:george after:2025-01-01 cabbage
func ParseSearchQuery(query string) (*imap.SearchCriteria, string, error) {
	criteria := imap.NewSearchCriteria()
	folder := ""

	if query == "" {
		return criteria, folder, nil
	}

	// Tokenize query, respecting quoted strings
	tokens := tokenizeQuery(query)

	// Track if we've seen folder: or label:
	folderFound := false
	var plainTextParts []string

	for _, token := range tokens {
		handled, extractedFolder, err := parseFilterToken(token, criteria, &folderFound)
		if err != nil {
			return nil, "", err
		}
		if handled {
			if extractedFolder != "" {
				folder = extractedFolder
			}
			continue
		}

		// Plain text - add to text search
		plainTextParts = append(plainTextParts, token)
	}

	// If we have plain text parts, add them to Text criteria
	if len(plainTextParts) > 0 {
		criteria.Text = []string{strings.Join(plainTextParts, " ")}
	}

	return criteria, folder, nil
}

// tokenizeQuery splits a query into tokens, respecting quoted strings.
// Handles quoted strings (e.g., "John Doe") and combines filter prefixes with quoted values
// (e.g., from:"John Doe" becomes a single token).
func tokenizeQuery(query string) []string {
	var tokens []string
	var current strings.Builder
	inQuotes := false

	for i, r := range query {
		if r == '"' {
			if inQuotes {
				// End of quoted string
				if current.Len() > 0 {
					tokens = append(tokens, `"`+current.String()+`"`)
					current.Reset()
				}
				inQuotes = false
			} else {
				// Start of quoted string
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				inQuotes = true
			}
		} else if r == ' ' && !inQuotes {
			// Space outside quotes - end of token
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(r)
		}

		// Handle last token
		if i == len(query)-1 && current.Len() > 0 {
			tokens = append(tokens, current.String())
		}
	}

	// Post-process: combine filter prefixes with immediately following quoted strings
	// e.g., "from:" and "\"John Doe\"" should become "from:\"John Doe\""
	var combinedTokens []string
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		// Check if this token ends with : and next token starts with "
		if strings.HasSuffix(token, ":") && i+1 < len(tokens) && strings.HasPrefix(tokens[i+1], `"`) {
			combinedTokens = append(combinedTokens, token+tokens[i+1])
			i++ // Skip next token as we've combined it
		} else {
			combinedTokens = append(combinedTokens, token)
		}
	}

	return combinedTokens
}

// unquote removes surrounding quotes from a string if present.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// parseDate parses a date string in YYYY-MM-DD format.
func parseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	// Try YYYY-MM-DD format
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: expected YYYY-MM-DD, got %s", dateStr)
	}
	return date, nil
}

// parseFolderFromQuery extracts folder name from query and returns folder and cleaned query.
// This is a legacy function kept for backward compatibility during Phase 1.
// Phase 2 uses ParseSearchQuery instead.
func parseFolderFromQuery(query string) (string, string) {
	folder := "INBOX"
	queryLower := strings.ToLower(query)
	if !strings.Contains(queryLower, "folder:") {
		return folder, query
	}

	parts := strings.Fields(queryLower)
	for i, part := range parts {
		if strings.HasPrefix(part, "folder:") {
			folder = strings.TrimPrefix(part, "folder:")
			// Remove folder: part from query
			queryParts := strings.Fields(query)
			if i < len(queryParts) {
				newParts := make([]string, 0, len(queryParts)-1)
				newParts = append(newParts, queryParts[:i]...)
				newParts = append(newParts, queryParts[i+1:]...)
				query = strings.Join(newParts, " ")
			}
			break
		}
	}
	return folder, query
}

// buildThreadMapFromMessages processes IMAP messages and builds a map of threads.
// Returns a map from stable thread ID to thread, and a map from stable thread ID to latest sent_at time.
// Messages without Message-ID headers or not found in the database are skipped with warnings.
func (s *Service) buildThreadMapFromMessages(ctx context.Context, userID string, messages []*imap.Message) (map[string]*models.Thread, map[string]*time.Time, error) {
	threadMap := make(map[string]*models.Thread)
	threadToLatestSentAt := make(map[string]*time.Time)

	for _, imapMsg := range messages {
		if imapMsg.Envelope == nil || len(imapMsg.Envelope.MessageId) == 0 {
			log.Printf("Warning: Message UID %d has no Message-ID, skipping", imapMsg.Uid)
			continue
		}

		messageID := imapMsg.Envelope.MessageId

		msg, err := db.GetMessageByMessageID(ctx, s.dbPool, userID, messageID)
		if err != nil {
			if errors.Is(err, db.ErrMessageNotFound) {
				log.Printf("Warning: Message with Message-ID %s not found in DB, skipping", messageID)
				continue
			}
			return nil, nil, fmt.Errorf("failed to get message from DB: %w", err)
		}

		thread, err := db.GetThreadByID(ctx, s.dbPool, msg.ThreadID)
		if err != nil {
			log.Printf("Warning: Failed to get thread %s: %v", msg.ThreadID, err)
			continue
		}

		if _, exists := threadMap[thread.StableThreadID]; !exists {
			threadMap[thread.StableThreadID] = thread
		}

		if msg.SentAt != nil {
			existingLatest := threadToLatestSentAt[thread.StableThreadID]
			if existingLatest == nil || msg.SentAt.After(*existingLatest) {
				threadToLatestSentAt[thread.StableThreadID] = msg.SentAt
			}
		}
	}

	return threadMap, threadToLatestSentAt, nil
}

// sortAndPaginateThreads sorts threads by latest sent_at (newest first) and applies pagination.
// Threads without sent_at are sorted to the end. Returns the paginated threads and total count.
func sortAndPaginateThreads(threadMap map[string]*models.Thread, threadToLatestSentAt map[string]*time.Time, page, limit int) ([]*models.Thread, int) {
	threads := make([]*models.Thread, 0, len(threadMap))
	for _, thread := range threadMap {
		threads = append(threads, thread)
	}

	sort.Slice(threads, func(i, j int) bool {
		sentAtI := threadToLatestSentAt[threads[i].StableThreadID]
		sentAtJ := threadToLatestSentAt[threads[j].StableThreadID]

		if sentAtI == nil {
			return false
		}
		if sentAtJ == nil {
			return true
		}
		return sentAtI.After(*sentAtJ)
	})

	totalCount := len(threads)
	offset := (page - 1) * limit
	if offset >= len(threads) {
		return nil, totalCount
	}

	end := offset + limit
	if end > len(threads) {
		end = len(threads)
	}

	return threads[offset:end], totalCount
}

// Search searches for threads matching the query in the specified folder.
// Supports Gmail-like syntax via ParseSearchQuery (from:, to:, subject:, after:, before:, folder:, label:).
// If no folder is specified in the query, defaults to INBOX.
// Returns threads sorted by latest sent_at (newest first), total count, and error.
// Note: Error handling tests for getClientAndSelectFolder, UidSearch, and FetchMessageHeaders
// require complex IMAP server mocking and are covered through integration tests.
func (s *Service) Search(ctx context.Context, userID string, query string, page, limit int) ([]*models.Thread, int, error) {
	// Parse the query using Gmail-like syntax
	criteria, extractedFolder, err := ParseSearchQuery(query)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrInvalidSearchQuery, err)
	}

	// Use extracted folder or default to INBOX
	folder := extractedFolder
	if folder == "" {
		folder = "INBOX"
	}

	var threads []*models.Thread
	var totalCount int

	err = s.withClientAndSelectFolder(ctx, userID, folder, func(client *imapclient.Client, _ *imap.MailboxStatus) error {
		uids, err := client.UidSearch(criteria)
		if err != nil {
			return fmt.Errorf("failed to search IMAP: %w", err)
		}

		if len(uids) == 0 {
			threads = nil
			totalCount = 0
			return nil
		}

		messages, err := FetchMessageHeaders(client, uids)
		if err != nil {
			return fmt.Errorf("failed to fetch message headers: %w", err)
		}

		threadMap, threadToLatestSentAt, err := s.buildThreadMapFromMessages(ctx, userID, messages)
		if err != nil {
			return err
		}

		threads, totalCount = sortAndPaginateThreads(threadMap, threadToLatestSentAt, page, limit)

		// Enrich threads with first message's from_address for display
		if err := db.EnrichThreadsWithFirstMessageFromAddress(ctx, s.dbPool, threads); err != nil {
			log.Printf("Warning: Failed to enrich threads with first message from address: %v", err)
			// Continue anyway - threads will work without the from_address
		}

		return nil
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get IMAP client: %w", err)
	}

	return threads, totalCount, nil
}
