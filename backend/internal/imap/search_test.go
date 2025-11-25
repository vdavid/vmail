package imap

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestParseFolderFromQuery(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedFolder string
		expectedQuery  string
	}{
		{
			name:           "returns INBOX and original query when no folder: prefix",
			query:          "test query",
			expectedFolder: "INBOX",
			expectedQuery:  "test query",
		},
		{
			name:           "extracts folder name from query",
			query:          "folder:Sent test",
			expectedFolder: "sent",
			expectedQuery:  "test",
		},
		{
			name:           "handles folder: at start",
			query:          "folder:Archive",
			expectedFolder: "archive",
			expectedQuery:  "",
		},
		{
			name:           "handles folder: in middle",
			query:          "test folder:Inbox query",
			expectedFolder: "inbox",
			expectedQuery:  "test query",
		},
		{
			name:           "handles multiple folder: occurrences (takes first)",
			query:          "folder:Sent test folder:Archive",
			expectedFolder: "sent",
			expectedQuery:  "test folder:Archive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			folder, query := parseFolderFromQuery(tt.query)
			assert.Equal(t, tt.expectedFolder, folder)
			assert.Equal(t, tt.expectedQuery, query)
		})
	}
}

func TestParseSearchQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		checkResult func(*testing.T, *imap.SearchCriteria, string)
	}{
		{
			name:        "handles empty query",
			query:       "",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Empty(t, folder)
				assert.NotNil(t, criteria)
			},
		},
		{
			name:        "parses from: filter",
			query:       "from:george",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Empty(t, folder)
				assert.Equal(t, "george", criteria.Header.Get("From"))
			},
		},
		{
			name:        "parses to: filter",
			query:       "to:alice",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Equal(t, "alice", criteria.Header.Get("To"))
			},
		},
		{
			name:        "parses subject: filter",
			query:       "subject:meeting",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Equal(t, "meeting", criteria.Header.Get("Subject"))
			},
		},
		{
			name:        "parses after: date filter",
			query:       "after:2025-01-01",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				expectedDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				assert.True(t, criteria.Since.Equal(expectedDate))
			},
		},
		{
			name:        "parses before: date filter",
			query:       "before:2025-12-31",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				expectedDate := time.Date(2025, 12, 31, 23, 59, 59, 999999999, time.UTC)
				assert.True(t, criteria.Before.Equal(expectedDate))
			},
		},
		{
			name:        "parses folder: filter",
			query:       "folder:Inbox",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Equal(t, "Inbox", folder)
				assert.Nil(t, criteria.Text)
			},
		},
		{
			name:        "parses label: filter (alias for folder)",
			query:       "label:Sent",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Equal(t, "Sent", folder)
			},
		},
		{
			name:        "parses plain text",
			query:       "cabbage",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Empty(t, folder)
				assert.Len(t, criteria.Text, 1)
				assert.Equal(t, "cabbage", criteria.Text[0])
			},
		},
		{
			name:        "parses multiple filters",
			query:       "from:george after:2025-01-01 cabbage",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Empty(t, folder)
				assert.Equal(t, "george", criteria.Header.Get("From"))
				expectedDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				assert.True(t, criteria.Since.Equal(expectedDate))
				assert.Len(t, criteria.Text, 1)
				assert.Equal(t, "cabbage", criteria.Text[0])
			},
		},
		{
			name:        "parses quoted strings",
			query:       `from:"John Doe"`,
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Equal(t, "John Doe", criteria.Header.Get("From"))
			},
		},
		{
			name:        "returns error for empty from: value",
			query:       "from:",
			expectError: true,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				// Error case, no need to check result
			},
		},
		{
			name:        "returns error for invalid date format",
			query:       "after:invalid-date",
			expectError: true,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				// Error case, no need to check result
			},
		},
		{
			name:        "folder: takes precedence over label:",
			query:       "folder:Inbox label:Sent",
			expectError: false,
			checkResult: func(t *testing.T, criteria *imap.SearchCriteria, folder string) {
				assert.Equal(t, "Inbox", folder)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			criteria, folder, err := ParseSearchQuery(tt.query)
			if tt.expectError {
				assert.Error(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, criteria, folder)
				}
				return
			}
			assert.NoError(t, err)
			if tt.checkResult != nil {
				tt.checkResult(t, criteria, folder)
			}
		})
	}
}

func TestSortAndPaginateThreads(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		threadMap   map[string]*models.Thread
		sentAtMap   map[string]*time.Time
		offset      int
		limit       int
		checkResult func(*testing.T, []*models.Thread, int)
	}{
		{
			name:      "handles empty thread map",
			threadMap: make(map[string]*models.Thread),
			sentAtMap: make(map[string]*time.Time),
			offset:    1,
			limit:     100,
			checkResult: func(t *testing.T, threads []*models.Thread, count int) {
				assert.Empty(t, threads)
				assert.Zero(t, count)
			},
		},
		{
			name: "handles pagination boundaries",
			threadMap: map[string]*models.Thread{
				"thread-1": {StableThreadID: "thread-1"},
				"thread-2": {StableThreadID: "thread-2"},
			},
			sentAtMap: map[string]*time.Time{
				"thread-1": &now,
				"thread-2": &now,
			},
			offset: 10,
			limit:  100,
			checkResult: func(t *testing.T, threads []*models.Thread, count int) {
				assert.Empty(t, threads, "should return empty when offset >= len")
				assert.Equal(t, 2, count)
			},
		},
		{
			name: "handles threads with nil sent_at",
			threadMap: map[string]*models.Thread{
				"thread-1": {StableThreadID: "thread-1"},
				"thread-2": {StableThreadID: "thread-2"},
			},
			sentAtMap: map[string]*time.Time{
				"thread-1": &now,
				"thread-2": nil,
			},
			offset: 1,
			limit:  100,
			checkResult: func(t *testing.T, threads []*models.Thread, count int) {
				assert.Len(t, threads, 2)
				assert.Equal(t, "thread-1", threads[0].StableThreadID, "thread with sent_at should come first")
				assert.Equal(t, 2, count)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threads, count := sortAndPaginateThreads(tt.threadMap, tt.sentAtMap, tt.offset, tt.limit)
			tt.checkResult(t, threads, count)
		})
	}
}

func TestTokenizeQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		checkResult func(*testing.T, []string)
	}{
		{
			name:  "handles unclosed quotes",
			query: `from:"John Doe`,
			checkResult: func(t *testing.T, tokens []string) {
				assert.NotEmpty(t, tokens, "should have at least one token")
				found := false
				for _, token := range tokens {
					if strings.Contains(token, "John Doe") {
						found = true
					}
				}
				assert.True(t, found, "token should contain 'John Doe'")
			},
		},
		{
			name:  "handles empty quoted strings",
			query: `from:"" test`,
			checkResult: func(t *testing.T, tokens []string) {
				assert.Len(t, tokens, 2, "should have 2 tokens (from: and test)")
				assert.Equal(t, "from:", tokens[0])
				assert.Equal(t, "test", tokens[1])
			},
		},
		{
			name:  "handles multiple spaces between tokens",
			query: "from:george    to:alice",
			checkResult: func(t *testing.T, tokens []string) {
				assert.Len(t, tokens, 2)
				assert.Equal(t, "from:george", tokens[0])
				assert.Equal(t, "to:alice", tokens[1])
			},
		},
		{
			name:  "handles nested quotes (quotes inside quotes)",
			query: `from:"John "Doe" Smith"`,
			checkResult: func(t *testing.T, tokens []string) {
				assert.NotEmpty(t, tokens, "should have at least one token")
			},
		},
		{
			name:  "handles quoted strings with spaces",
			query: `from:"John Doe" test`,
			checkResult: func(t *testing.T, tokens []string) {
				assert.Len(t, tokens, 2)
				found := false
				for _, token := range tokens {
					if strings.Contains(token, "John Doe") {
						found = true
					}
				}
				assert.True(t, found, "token should contain 'John Doe'")
			},
		},
		{
			name:  "handles filter prefix with quoted value",
			query: `from: "John Doe"`,
			checkResult: func(t *testing.T, tokens []string) {
				assert.NotEmpty(t, tokens)
				found := false
				for _, token := range tokens {
					if strings.Contains(token, "from:") && strings.Contains(token, "John Doe") {
						found = true
					}
				}
				assert.True(t, found, "from: and 'John Doe' should be combined")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tokenizeQuery(tt.query)
			tt.checkResult(t, tokens)
		})
	}
}

func TestService_buildThreadMapFromMessages(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := testutil.GetTestEncryptor(t)
	service := NewService(pool, NewPool(), encryptor)
	defer service.Close()

	ctx := context.Background()
	userID, err := db.GetOrCreateUser(ctx, pool, "build-thread-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	tests := []struct {
		name        string
		setup       func() []*imap.Message
		expectError bool
		checkResult func(*testing.T, map[string]*models.Thread, map[string]*time.Time)
	}{
		{
			name: "returns error when GetMessageByMessageID returns non-NotFound error",
			setup: func() []*imap.Message {
				return []*imap.Message{
					{
						Uid: 1,
						Envelope: &imap.Envelope{
							MessageId: "<test-message@example.com>",
						},
					},
				}
			},
			expectError: true,
			checkResult: func(t *testing.T, threadMap map[string]*models.Thread, sentAtMap map[string]*time.Time) {
				// Error case, no need to check result
			},
		},
		{
			name: "continues gracefully when GetThreadByID returns error",
			setup: func() []*imap.Message {
				messageID := "<thread-error-test@example.com>"
				thread := &models.Thread{
					UserID:         userID,
					StableThreadID: messageID,
					Subject:        "Test Thread",
				}
				if err := db.SaveThread(ctx, pool, thread); err != nil {
					t.Fatalf("Failed to save thread: %v", err)
				}

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

				return []*imap.Message{
					{
						Uid: 1,
						Envelope: &imap.Envelope{
							MessageId: messageID,
						},
					},
				}
			},
			expectError: false,
			checkResult: func(t *testing.T, threadMap map[string]*models.Thread, sentAtMap map[string]*time.Time) {
				assert.Empty(t, threadMap, "thread map should be empty when thread was deleted")
				assert.Empty(t, sentAtMap)
			},
		},
		{
			name: "skips messages not found in database",
			setup: func() []*imap.Message {
				return []*imap.Message{
					{
						Uid: 999,
						Envelope: &imap.Envelope{
							MessageId: "<non-existent@example.com>",
						},
					},
				}
			},
			expectError: false,
			checkResult: func(t *testing.T, threadMap map[string]*models.Thread, sentAtMap map[string]*time.Time) {
				assert.Empty(t, threadMap)
				assert.Empty(t, sentAtMap)
			},
		},
		{
			name: "skips messages without Message-ID",
			setup: func() []*imap.Message {
				return []*imap.Message{
					{
						Uid:      1,
						Envelope: &imap.Envelope{},
					},
				}
			},
			expectError: false,
			checkResult: func(t *testing.T, threadMap map[string]*models.Thread, sentAtMap map[string]*time.Time) {
				assert.Empty(t, threadMap)
				assert.Empty(t, sentAtMap)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := tt.setup()
			var err error
			var threadMap map[string]*models.Thread
			var sentAtMap map[string]*time.Time

			if tt.name == "returns error when GetMessageByMessageID returns non-NotFound error" {
				canceledCtx, cancel := context.WithCancel(ctx)
				cancel()
				threadMap, sentAtMap, err = service.buildThreadMapFromMessages(canceledCtx, userID, messages)
			} else {
				threadMap, sentAtMap, err = service.buildThreadMapFromMessages(ctx, userID, messages)
			}

			if tt.expectError {
				assert.Error(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, threadMap, sentAtMap)
				}
				return
			}
			assert.NoError(t, err)
			if tt.checkResult != nil {
				tt.checkResult(t, threadMap, sentAtMap)
			}
		})
	}
}
