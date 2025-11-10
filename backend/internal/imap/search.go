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
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
)

// parseFilterToken processes a single token and updates criteria/folder accordingly.
// Returns (handled, folder, error) where handled indicates if the token was a filter.
func parseFilterToken(token string, criteria *imap.SearchCriteria, folderFound *bool) (bool, string, error) {
	// Check for filter without value
	if strings.HasSuffix(token, ":") {
		return false, "", fmt.Errorf("empty filter value: %s", token)
	}

	// Check for from: filter
	if strings.HasPrefix(token, "from:") {
		value := strings.TrimPrefix(token, "from:")
		if value == "" {
			return false, "", fmt.Errorf("empty from: value")
		}
		criteria.Header.Add("From", unquote(value))
		return true, "", nil
	}

	// Check for to: filter
	if strings.HasPrefix(token, "to:") {
		value := strings.TrimPrefix(token, "to:")
		if value == "" {
			return false, "", fmt.Errorf("empty to: value")
		}
		criteria.Header.Add("To", unquote(value))
		return true, "", nil
	}

	// Check for subject: filter
	if strings.HasPrefix(token, "subject:") {
		value := strings.TrimPrefix(token, "subject:")
		if value == "" {
			return false, "", fmt.Errorf("empty subject: value")
		}
		criteria.Header.Add("Subject", unquote(value))
		return true, "", nil
	}

	// Check for after: filter
	if strings.HasPrefix(token, "after:") {
		value := strings.TrimPrefix(token, "after:")
		if value == "" {
			return false, "", fmt.Errorf("empty after: value")
		}
		date, err := parseDate(value)
		if err != nil {
			return false, "", fmt.Errorf("invalid date format for after: %w", err)
		}
		criteria.Since = date
		return true, "", nil
	}

	// Check for before: filter
	if strings.HasPrefix(token, "before:") {
		value := strings.TrimPrefix(token, "before:")
		if value == "" {
			return false, "", fmt.Errorf("empty before: value")
		}
		date, err := parseDate(value)
		if err != nil {
			return false, "", fmt.Errorf("invalid date format for before: %w", err)
		}
		// Set to end of day
		date = time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())
		criteria.Before = date
		return true, "", nil
	}

	// Check for folder: or label: filter
	if strings.HasPrefix(token, "folder:") || strings.HasPrefix(token, "label:") {
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
	plainTextParts := []string{}

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

	return tokens
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
func (s *Service) buildThreadMapFromMessages(ctx context.Context, userID string, messages []*imap.Message) (map[string]*models.Thread, map[string]*time.Time, error) {
	threadMap := make(map[string]*models.Thread)
	threadToLatestSentAt := make(map[string]*time.Time)

	for _, imapMsg := range messages {
		if imapMsg.Envelope == nil || len(imapMsg.Envelope.MessageId) == 0 {
			log.Printf("Warning: Message UID %d has no Message-ID, skipping", imapMsg.Uid)
			continue
		}

		messageID := imapMsg.Envelope.MessageId

		msg, err := db.GetMessageByMessageID(ctx, s.pool, userID, messageID)
		if err != nil {
			if errors.Is(err, db.ErrMessageNotFound) {
				log.Printf("Warning: Message with Message-ID %s not found in DB, skipping", messageID)
				continue
			}
			return nil, nil, fmt.Errorf("failed to get message from DB: %w", err)
		}

		thread, err := db.GetThreadByID(ctx, s.pool, msg.ThreadID)
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

// sortAndPaginateThreads sorts threads by latest sent_at and applies pagination.
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
		return []*models.Thread{}, totalCount
	}

	end := offset + limit
	if end > len(threads) {
		end = len(threads)
	}

	return threads[offset:end], totalCount
}

// Search searches for threads matching the query in the specified folder.
// Supports Gmail-like syntax via ParseSearchQuery.
// Returns threads, total count, and error.
func (s *Service) Search(ctx context.Context, userID string, query string, page, limit int) ([]*models.Thread, int, error) {
	// Parse the query using Gmail-like syntax
	criteria, extractedFolder, err := ParseSearchQuery(query)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid search query: %w", err)
	}

	// Use extracted folder or default to INBOX
	folder := extractedFolder
	if folder == "" {
		folder = "INBOX"
	}

	client, _, err := s.getClientAndSelectFolder(ctx, userID, folder)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get IMAP client: %w", err)
	}

	uids, err := client.UidSearch(criteria)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search IMAP: %w", err)
	}

	if len(uids) == 0 {
		return []*models.Thread{}, 0, nil
	}

	messages, err := FetchMessageHeaders(client, uids)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch message headers: %w", err)
	}

	threadMap, threadToLatestSentAt, err := s.buildThreadMapFromMessages(ctx, userID, messages)
	if err != nil {
		return nil, 0, err
	}

	threads, totalCount := sortAndPaginateThreads(threadMap, threadToLatestSentAt, page, limit)
	return threads, totalCount, nil
}
