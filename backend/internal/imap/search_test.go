package imap

import (
	"strings"
	"testing"
	"time"
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
