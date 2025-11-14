package imap

import (
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestRunThreadCommand(t *testing.T) {
	t.Run("returns error for nil client", func(t *testing.T) {
		_, err := RunThreadCommand(nil)
		if err == nil {
			t.Error("Expected error for nil client")
		}
		if err.Error() != "client is nil" {
			t.Errorf("Expected error 'client is nil', got: %v", err)
		}
	})

	t.Run("handles empty mailbox", func(t *testing.T) {
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		server.EnsureINBOX(t)

		client, cleanup := server.Connect(t)
		defer cleanup()

		// Select INBOX (which is empty)
		_, err := client.Select("INBOX", false)
		if err != nil {
			t.Fatalf("Failed to select INBOX: %v", err)
		}

		// Check if server supports THREAD
		caps, err := client.Capability()
		if err != nil {
			t.Fatalf("Failed to check capabilities: %v", err)
		}

		// Run thread command on empty mailbox
		threads, err := RunThreadCommand(client)
		if !caps["THREAD"] {
			// Server doesn't support THREAD, expect an error
			if err == nil {
				t.Error("Expected error for server without THREAD support")
			}
			return
		}

		// Server supports THREAD, should succeed
		if err != nil {
			t.Fatalf("RunThreadCommand should succeed on empty mailbox: %v", err)
		}

		if threads == nil {
			t.Error("Expected empty slice, got nil")
		}
		if len(threads) != 0 {
			t.Errorf("Expected empty threads slice, got %d threads", len(threads))
		}
	})

	t.Run("handles mailbox with unthreaded messages", func(t *testing.T) {
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		server.EnsureINBOX(t)

		// Add some messages without threading relationships
		now := time.Now()
		server.AddMessage(t, "INBOX", "<msg1@test>", "Subject 1", "from@test.com", "to@test.com", now)
		server.AddMessage(t, "INBOX", "<msg2@test>", "Subject 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))

		client, cleanup := server.Connect(t)
		defer cleanup()

		_, err := client.Select("INBOX", false)
		if err != nil {
			t.Fatalf("Failed to select INBOX: %v", err)
		}

		// Run thread command
		threads, err := RunThreadCommand(client)
		if err != nil {
			// Some servers may not support THREAD command
			// In that case, we expect an error
			if err.Error() == "" {
				t.Error("Expected non-empty error message")
			}
			return
		}

		// If successful, we should have threads (possibly one per message if unthreaded)
		if threads == nil {
			t.Error("Expected threads slice, got nil")
		}
		// Unthreaded messages might be returned as separate threads or as a single thread
		// The exact behavior depends on the server implementation
	})

	t.Run("handles server without THREAD support", func(t *testing.T) {
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		server.EnsureINBOX(t)

		client, cleanup := server.Connect(t)
		defer cleanup()

		// Check if server supports THREAD
		caps, err := client.Capability()
		if err != nil {
			t.Fatalf("Failed to check capabilities: %v", err)
		}

		// The memory backend may or may not support THREAD
		// If it doesn't, we should get an error
		if !caps["THREAD"] {
			_, err := RunThreadCommand(client)
			if err == nil {
				t.Error("Expected error for server without THREAD support")
			}
		} else {
			// Server supports THREAD, so test should pass
			_, err := RunThreadCommand(client)
			if err != nil {
				t.Fatalf("RunThreadCommand should succeed when THREAD is supported: %v", err)
			}
		}
	})

	t.Run("handles network errors during thread command", func(t *testing.T) {
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		client, _ := server.Connect(t)
		// Close the client to simulate network error
		_ = client.Logout()

		// Try to run thread command with closed client
		_, err := RunThreadCommand(client)
		if err == nil {
			t.Error("Expected error when client is closed")
		}
	})
}
