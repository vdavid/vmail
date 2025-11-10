package imap

import (
	"testing"
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
