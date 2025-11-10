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

// parseFolderFromQuery extracts folder name from query and returns folder and cleaned query.
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
// For Phase 1, this handles basic text search only.
// Returns threads, total count, and error.
func (s *Service) Search(ctx context.Context, userID string, query string, page, limit int) ([]*models.Thread, int, error) {
	folder, cleanedQuery := parseFolderFromQuery(query)

	client, _, err := s.getClientAndSelectFolder(ctx, userID, folder)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get IMAP client: %w", err)
	}

	criteria := imap.NewSearchCriteria()
	if cleanedQuery != "" {
		criteria.Text = []string{cleanedQuery}
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
