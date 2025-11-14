package imap

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestParseFolderFromQuery(t *testing.T) {
	t.Run("returns INBOX and original query when no folder: prefix", func(t *testing.T) {
		folder, query := parseFolderFromQuery("test query")
		if folder != "INBOX" {
			t.Errorf("Expected folder 'INBOX', got '%s'", folder)
		}
		if query != "test query" {
			t.Errorf("Expected query 'test query', got '%s'", query)
		}
	})

	t.Run("extracts folder name from query", func(t *testing.T) {
		folder, query := parseFolderFromQuery("folder:Sent test")
		if folder != "sent" {
			t.Errorf("Expected folder 'sent', got '%s'", folder)
		}
		if query != "test" {
			t.Errorf("Expected query 'test', got '%s'", query)
		}
	})

	t.Run("handles folder: at start", func(t *testing.T) {
		folder, query := parseFolderFromQuery("folder:Archive")
		if folder != "archive" {
			t.Errorf("Expected folder 'archive', got '%s'", folder)
		}
		if query != "" {
			t.Errorf("Expected empty query, got '%s'", query)
		}
	})

	t.Run("handles folder: in middle", func(t *testing.T) {
		folder, query := parseFolderFromQuery("test folder:Inbox query")
		if folder != "inbox" {
			t.Errorf("Expected folder 'inbox', got '%s'", folder)
		}
		if query != "test query" {
			t.Errorf("Expected query 'test query', got '%s'", query)
		}
	})

	t.Run("handles multiple folder: occurrences (takes first)", func(t *testing.T) {
		folder, query := parseFolderFromQuery("folder:Sent test folder:Archive")
		if folder != "sent" {
			t.Errorf("Expected folder 'sent', got '%s'", folder)
		}
		if query != "test folder:Archive" {
			t.Errorf("Expected query 'test folder:Archive', got '%s'", query)
		}
	})
}

func TestParseSearchQuery(t *testing.T) {
	t.Run("handles empty query", func(t *testing.T) {
		criteria, folder, err := ParseSearchQuery("")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if folder != "" {
			t.Errorf("Expected empty folder, got '%s'", folder)
		}
		if criteria == nil {
			t.Error("Expected criteria to be non-nil")
		}
	})

	t.Run("parses from: filter", func(t *testing.T) {
		criteria, folder, err := ParseSearchQuery("from:george")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if folder != "" {
			t.Errorf("Expected empty folder, got '%s'", folder)
		}
		if criteria.Header.Get("From") != "george" {
			t.Errorf("Expected From header 'george', got '%s'", criteria.Header.Get("From"))
		}
	})

	t.Run("parses to: filter", func(t *testing.T) {
		criteria, _, err := ParseSearchQuery("to:alice")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if criteria.Header.Get("To") != "alice" {
			t.Errorf("Expected To header 'alice', got '%s'", criteria.Header.Get("To"))
		}
	})

	t.Run("parses subject: filter", func(t *testing.T) {
		criteria, _, err := ParseSearchQuery("subject:meeting")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if criteria.Header.Get("Subject") != "meeting" {
			t.Errorf("Expected Subject header 'meeting', got '%s'", criteria.Header.Get("Subject"))
		}
	})

	t.Run("parses after: date filter", func(t *testing.T) {
		criteria, _, err := ParseSearchQuery("after:2025-01-01")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		expectedDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		if !criteria.Since.Equal(expectedDate) {
			t.Errorf("Expected Since date %v, got %v", expectedDate, criteria.Since)
		}
	})

	t.Run("parses before: date filter", func(t *testing.T) {
		criteria, _, err := ParseSearchQuery("before:2025-12-31")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		expectedDate := time.Date(2025, 12, 31, 23, 59, 59, 999999999, time.UTC)
		if !criteria.Before.Equal(expectedDate) {
			t.Errorf("Expected Before date %v, got %v", expectedDate, criteria.Before)
		}
	})

	t.Run("parses folder: filter", func(t *testing.T) {
		criteria, folder, err := ParseSearchQuery("folder:Inbox")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if folder != "Inbox" {
			t.Errorf("Expected folder 'Inbox', got '%s'", folder)
		}
		if criteria.Text != nil {
			t.Error("Expected no text criteria when folder is specified")
		}
	})

	t.Run("parses label: filter (alias for folder)", func(t *testing.T) {
		_, folder, err := ParseSearchQuery("label:Sent")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if folder != "Sent" {
			t.Errorf("Expected folder 'Sent', got '%s'", folder)
		}
	})

	t.Run("parses plain text", func(t *testing.T) {
		criteria, folder, err := ParseSearchQuery("cabbage")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if folder != "" {
			t.Errorf("Expected empty folder, got '%s'", folder)
		}
		if len(criteria.Text) != 1 || criteria.Text[0] != "cabbage" {
			t.Errorf("Expected text 'cabbage', got %v", criteria.Text)
		}
	})

	t.Run("parses multiple filters", func(t *testing.T) {
		criteria, folder, err := ParseSearchQuery("from:george after:2025-01-01 cabbage")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if folder != "" {
			t.Errorf("Expected empty folder, got '%s'", folder)
		}
		if criteria.Header.Get("From") != "george" {
			t.Errorf("Expected From header 'george', got '%s'", criteria.Header.Get("From"))
		}
		expectedDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		if !criteria.Since.Equal(expectedDate) {
			t.Errorf("Expected Since date %v, got %v", expectedDate, criteria.Since)
		}
		if len(criteria.Text) != 1 || criteria.Text[0] != "cabbage" {
			t.Errorf("Expected text 'cabbage', got %v", criteria.Text)
		}
	})

	t.Run("parses quoted strings", func(t *testing.T) {
		criteria, _, err := ParseSearchQuery(`from:"John Doe"`)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if criteria.Header.Get("From") != "John Doe" {
			t.Errorf("Expected From header 'John Doe', got '%s'", criteria.Header.Get("From"))
		}
	})

	t.Run("returns error for empty from: value", func(t *testing.T) {
		_, _, err := ParseSearchQuery("from:")
		if err == nil {
			t.Error("Expected error for empty from: value")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("Expected error message about empty value, got %v", err)
		}
	})

	t.Run("returns error for invalid date format", func(t *testing.T) {
		_, _, err := ParseSearchQuery("after:invalid-date")
		if err == nil {
			t.Error("Expected error for invalid date format")
		}
		if !strings.Contains(err.Error(), "invalid date format") {
			t.Errorf("Expected error message about invalid date format, got %v", err)
		}
	})

	t.Run("folder: takes precedence over label:", func(t *testing.T) {
		_, folder, err := ParseSearchQuery("folder:Inbox label:Sent")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if folder != "Inbox" {
			t.Errorf("Expected folder 'Inbox' (folder: takes precedence), got '%s'", folder)
		}
	})
}

func TestSortAndPaginateThreads(t *testing.T) {
	t.Run("handles empty thread map", func(t *testing.T) {
		threadMap := make(map[string]*models.Thread)
		sentAtMap := make(map[string]*time.Time)

		threads, count := sortAndPaginateThreads(threadMap, sentAtMap, 1, 100)
		if len(threads) != 0 {
			t.Errorf("Expected 0 threads, got %d", len(threads))
		}
		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}
	})

	t.Run("handles pagination boundaries", func(t *testing.T) {
		threadMap := map[string]*models.Thread{
			"thread-1": {StableThreadID: "thread-1"},
			"thread-2": {StableThreadID: "thread-2"},
		}
		now := time.Now()
		sentAtMap := map[string]*time.Time{
			"thread-1": &now,
			"thread-2": &now,
		}

		// Test offset >= len(threads)
		threads, count := sortAndPaginateThreads(threadMap, sentAtMap, 10, 100)
		if len(threads) != 0 {
			t.Errorf("Expected 0 threads when offset >= len, got %d", len(threads))
		}
		if count != 2 {
			t.Errorf("Expected total count 2, got %d", count)
		}

		// Test end > len(threads)
		threads, count = sortAndPaginateThreads(threadMap, sentAtMap, 1, 100)
		if len(threads) != 2 {
			t.Errorf("Expected 2 threads when limit > len, got %d", len(threads))
		}
		if count != 2 {
			t.Errorf("Expected total count 2, got %d", count)
		}
	})

	t.Run("handles threads with nil sent_at", func(t *testing.T) {
		threadMap := map[string]*models.Thread{
			"thread-1": {StableThreadID: "thread-1"},
			"thread-2": {StableThreadID: "thread-2"},
		}
		now := time.Now()
		sentAtMap := map[string]*time.Time{
			"thread-1": &now,
			"thread-2": nil, // No sent_at
		}

		threads, count := sortAndPaginateThreads(threadMap, sentAtMap, 1, 100)
		if len(threads) != 2 {
			t.Errorf("Expected 2 threads, got %d", len(threads))
		}
		// Thread with sent_at should come first
		if threads[0].StableThreadID != "thread-1" {
			t.Errorf("Expected thread-1 first (has sent_at), got %s", threads[0].StableThreadID)
		}
		if count != 2 {
			t.Errorf("Expected total count 2, got %d", count)
		}
	})
}

func TestTokenizeQuery(t *testing.T) {
	t.Run("handles unclosed quotes", func(t *testing.T) {
		// Unclosed quote should treat the rest as part of the token
		tokens := tokenizeQuery(`from:"John Doe`)
		// The tokenizer should handle this gracefully - the quote starts but never closes
		// So "John Doe" (without closing quote) should be part of the token
		if len(tokens) == 0 {
			t.Error("Expected at least one token for unclosed quote")
		}
		// Verify the behavior: the unclosed quote should be included in the token
		found := false
		for _, token := range tokens {
			if strings.Contains(token, "John Doe") {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected token to contain 'John Doe', got tokens: %v", tokens)
		}
	})

	t.Run("handles empty quoted strings", func(t *testing.T) {
		tokens := tokenizeQuery(`from:"" test`)
		// Empty quoted strings are skipped (not tokenized) - this is the current behavior
		// The tokenizer processes from: and test, skipping the empty quotes
		if len(tokens) != 2 {
			t.Errorf("Expected 2 tokens (from: and test), got %d: %v", len(tokens), tokens)
		}
		if tokens[0] != "from:" {
			t.Errorf("Expected first token 'from:', got '%s'", tokens[0])
		}
		if tokens[1] != "test" {
			t.Errorf("Expected second token 'test', got '%s'", tokens[1])
		}
	})

	t.Run("handles multiple spaces between tokens", func(t *testing.T) {
		tokens := tokenizeQuery("from:george    to:alice")
		// Multiple spaces should be collapsed (treated as single separator)
		if len(tokens) != 2 {
			t.Errorf("Expected 2 tokens, got %d: %v", len(tokens), tokens)
		}
		if tokens[0] != "from:george" {
			t.Errorf("Expected first token 'from:george', got '%s'", tokens[0])
		}
		if tokens[1] != "to:alice" {
			t.Errorf("Expected second token 'to:alice', got '%s'", tokens[1])
		}
	})

	t.Run("handles nested quotes (quotes inside quotes)", func(t *testing.T) {
		// The current implementation doesn't handle escaped quotes, but we test the behavior
		tokens := tokenizeQuery(`from:"John "Doe" Smith"`)
		// The tokenizer treats each quote as a toggle, so nested quotes will be tokenized
		// This is expected behavior - the tokenizer doesn't handle escaped quotes
		if len(tokens) == 0 {
			t.Error("Expected at least one token for nested quotes")
		}
	})

	t.Run("handles quoted strings with spaces", func(t *testing.T) {
		tokens := tokenizeQuery(`from:"John Doe" test`)
		if len(tokens) != 2 {
			t.Errorf("Expected 2 tokens, got %d: %v", len(tokens), tokens)
		}
		// The quoted string should be combined with the prefix if applicable
		// Check that "John Doe" is in one of the tokens
		found := false
		for _, token := range tokens {
			if strings.Contains(token, "John Doe") {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected token to contain 'John Doe', got tokens: %v", tokens)
		}
	})

	t.Run("handles filter prefix with quoted value", func(t *testing.T) {
		tokens := tokenizeQuery(`from: "John Doe"`)
		// The tokenizer should combine "from:" with the following quoted string
		if len(tokens) == 0 {
			t.Error("Expected at least one token")
		}
		// Check that from: and "John Doe" are combined
		found := false
		for _, token := range tokens {
			if strings.Contains(token, "from:") && strings.Contains(token, "John Doe") {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected 'from:' and 'John Doe' to be combined, got tokens: %v", tokens)
		}
	})
}

func TestService_buildThreadMapFromMessages(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := getTestEncryptorForSearch(t)
	service := NewService(pool, encryptor)
	defer service.Close()

	ctx := context.Background()
	userID, err := db.GetOrCreateUser(ctx, pool, "build-thread-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	t.Run("returns error when GetMessageByMessageID returns non-NotFound error", func(t *testing.T) {
		// Create a cancelled context to simulate a database error
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately to cause context error

		imapMsg := &imap.Message{
			Uid: 1,
			Envelope: &imap.Envelope{
				MessageId: "<test-message@example.com>",
			},
		}

		_, _, err := service.buildThreadMapFromMessages(cancelledCtx, userID, []*imap.Message{imapMsg})
		if err == nil {
			t.Error("Expected error when GetMessageByMessageID returns non-NotFound error")
		}
		if !strings.Contains(err.Error(), "failed to get message from DB") {
			t.Errorf("Expected error message about 'failed to get message from DB', got: %v", err)
		}
	})

	t.Run("continues gracefully when GetThreadByID returns error", func(t *testing.T) {
		// Create a thread and message
		messageID := "<thread-error-test@example.com>"
		thread := &models.Thread{
			UserID:         userID,
			StableThreadID: messageID,
			Subject:        "Test Thread",
		}
		if err := db.SaveThread(ctx, pool, thread); err != nil {
			t.Fatalf("Failed to save thread: %v", err)
		}

		// Create a message linked to this thread
		message := &models.Message{
			ThreadID:        thread.ID,
			UserID:          userID,
			IMAPUID:         1,
			IMAPFolderName:  "INBOX",
			MessageIDHeader: messageID,
			FromAddress:     "from@example.com",
			Subject:         "Test Subject",
		}
		if err := db.SaveMessage(ctx, pool, message); err != nil {
			t.Fatalf("Failed to save message: %v", err)
		}

		// Delete the thread to simulate GetThreadByID returning an error
		_, err := pool.Exec(ctx, "DELETE FROM threads WHERE id = $1", thread.ID)
		if err != nil {
			t.Fatalf("Failed to delete thread: %v", err)
		}

		// Now buildThreadMapFromMessages should skip this message and continue
		imapMsg := &imap.Message{
			Uid: 1,
			Envelope: &imap.Envelope{
				MessageId: messageID,
			},
		}

		threadMap, sentAtMap, err := service.buildThreadMapFromMessages(ctx, userID, []*imap.Message{imapMsg})
		if err != nil {
			t.Errorf("Expected no error (should skip message with missing thread), got: %v", err)
		}
		// The thread should not be in the map because GetThreadByID failed
		if len(threadMap) != 0 {
			t.Errorf("Expected empty thread map (thread was deleted), got: %v", threadMap)
		}
		if len(sentAtMap) != 0 {
			t.Errorf("Expected empty sentAt map, got: %v", sentAtMap)
		}
	})

	t.Run("skips messages not found in database", func(t *testing.T) {
		// Message that doesn't exist in DB
		imapMsg := &imap.Message{
			Uid: 999,
			Envelope: &imap.Envelope{
				MessageId: "<non-existent@example.com>",
			},
		}

		threadMap, sentAtMap, err := service.buildThreadMapFromMessages(ctx, userID, []*imap.Message{imapMsg})
		if err != nil {
			t.Errorf("Expected no error (should skip message not found), got: %v", err)
		}
		if len(threadMap) != 0 {
			t.Errorf("Expected empty thread map, got: %v", threadMap)
		}
		if len(sentAtMap) != 0 {
			t.Errorf("Expected empty sentAt map, got: %v", sentAtMap)
		}
	})

	t.Run("skips messages without Message-ID", func(t *testing.T) {
		imapMsg := &imap.Message{
			Uid:      1,
			Envelope: &imap.Envelope{
				// No MessageId
			},
		}

		threadMap, sentAtMap, err := service.buildThreadMapFromMessages(ctx, userID, []*imap.Message{imapMsg})
		if err != nil {
			t.Errorf("Expected no error (should skip message without Message-ID), got: %v", err)
		}
		if len(threadMap) != 0 {
			t.Errorf("Expected empty thread map, got: %v", threadMap)
		}
		if len(sentAtMap) != 0 {
			t.Errorf("Expected empty sentAt map, got: %v", sentAtMap)
		}
	})
}

func getTestEncryptorForSearch(t *testing.T) *crypto.Encryptor {
	t.Helper()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, err := crypto.NewEncryptor(base64Key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	return encryptor
}
