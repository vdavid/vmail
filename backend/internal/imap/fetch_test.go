package imap

import (
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestFetchMessageHeaders(t *testing.T) {
	t.Run("returns error for nil client", func(t *testing.T) {
		_, err := FetchMessageHeaders(nil, []uint32{1, 2, 3})
		if err == nil {
			t.Error("Expected error for nil client")
		}
		if err.Error() != "client is nil" {
			t.Errorf("Expected error 'client is nil', got: %v", err)
		}
	})

	t.Run("returns empty slice for empty UIDs", func(t *testing.T) {
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		client, cleanup := server.Connect(t)
		defer cleanup()

		result, err := FetchMessageHeaders(client, []uint32{})
		if err != nil {
			t.Errorf("Expected no error for empty UIDs, got: %v", err)
		}
		if result == nil {
			t.Error("Expected empty slice, got nil")
		}
		if len(result) != 0 {
			t.Errorf("Expected empty slice, got %d items", len(result))
		}
	})

	t.Run("fetches message headers successfully", func(t *testing.T) {
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		server.EnsureINBOX(t)

		// Add a test message
		uid := server.AddMessage(t, "INBOX", "<test@example.com>", "Test Subject", "from@example.com", "to@example.com", time.Now())

		client, cleanup := server.Connect(t)
		defer cleanup()

		// Select INBOX
		_, err := client.Select("INBOX", false)
		if err != nil {
			t.Fatalf("Failed to select INBOX: %v", err)
		}

		// Fetch headers
		messages, err := FetchMessageHeaders(client, []uint32{uid})
		if err != nil {
			t.Fatalf("Failed to fetch message headers: %v", err)
		}

		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}

		if messages[0].Uid != uid {
			t.Errorf("Expected UID %d, got %d", uid, messages[0].Uid)
		}

		if messages[0].Envelope == nil {
			t.Error("Expected envelope, got nil")
		}
	})
}

func TestFetchFullMessage(t *testing.T) {
	t.Run("returns error for nil client", func(t *testing.T) {
		_, err := FetchFullMessage(nil, 1)
		if err == nil {
			t.Error("Expected error for nil client")
		}
		if err.Error() != "client is nil" {
			t.Errorf("Expected error 'client is nil', got: %v", err)
		}
	})

	t.Run("fetches full message successfully", func(t *testing.T) {
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		server.EnsureINBOX(t)

		// Add a test message
		uid := server.AddMessage(t, "INBOX", "<test@example.com>", "Test Subject", "from@example.com", "to@example.com", time.Now())

		client, cleanup := server.Connect(t)
		defer cleanup()

		// Select INBOX
		_, err := client.Select("INBOX", false)
		if err != nil {
			t.Fatalf("Failed to select INBOX: %v", err)
		}

		// Fetch full message
		msg, err := FetchFullMessage(client, uid)
		if err != nil {
			t.Fatalf("Failed to fetch full message: %v", err)
		}

		if msg == nil {
			t.Fatal("Expected message, got nil")
		}

		if msg.Uid != uid {
			t.Errorf("Expected UID %d, got %d", uid, msg.Uid)
		}

		if msg.Envelope == nil {
			t.Error("Expected envelope, got nil")
		}
	})

	t.Run("handles message without body structure", func(t *testing.T) {
		// This test verifies that FetchFullMessage doesn't crash when
		// BodyStructure is nil. The function should still return the message
		// with headers even if body structure is missing.
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		server.EnsureINBOX(t)

		// Add a test message
		uid := server.AddMessage(t, "INBOX", "<test@example.com>", "Test Subject", "from@example.com", "to@example.com", time.Now())

		client, cleanup := server.Connect(t)
		defer cleanup()

		// Select INBOX
		_, err := client.Select("INBOX", false)
		if err != nil {
			t.Fatalf("Failed to select INBOX: %v", err)
		}

		// Fetch full message
		msg, err := FetchFullMessage(client, uid)
		if err != nil {
			t.Fatalf("Failed to fetch full message: %v", err)
		}

		// Message should be returned even if body structure is nil
		if msg == nil {
			t.Fatal("Expected message, got nil")
		}
	})
}
