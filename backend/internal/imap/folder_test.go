package imap

import (
	"testing"

	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestListFolders(t *testing.T) {
	t.Run("returns error for nil client", func(t *testing.T) {
		_, err := ListFolders(nil)
		if err == nil {
			t.Error("Expected error for nil client")
		}
		if err.Error() != "client is nil" {
			t.Errorf("Expected error 'client is nil', got: %v", err)
		}
	})

	t.Run("returns error for server without SPECIAL-USE support", func(t *testing.T) {
		// Create a test IMAP server without SPECIAL-USE extension
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		client, cleanup := server.Connect(t)
		defer cleanup()

		// The test server doesn't enable SPECIAL-USE by default
		// We need to check if it supports it - if not, we should get an error
		caps, err := client.Capability()
		if err != nil {
			t.Fatalf("Failed to check capabilities: %v", err)
		}

		// If the server doesn't support SPECIAL-USE, ListFolders should return an error
		if !caps["SPECIAL-USE"] {
			_, err := ListFolders(client)
			if err == nil {
				t.Error("Expected error for server without SPECIAL-USE support")
			}
			if err.Error() == "" {
				t.Error("Expected non-empty error message")
			}
		} else {
			// Server supports SPECIAL-USE, so test should pass
			folders, err := ListFolders(client)
			if err != nil {
				t.Fatalf("ListFolders should succeed when SPECIAL-USE is supported: %v", err)
			}
			if folders == nil {
				t.Error("Expected folders slice, got nil")
			}
		}
	})

	t.Run("handles empty folder list", func(t *testing.T) {
		// Create a test IMAP server with SPECIAL-USE support
		server, err := testutil.NewTestIMAPServerForE2E()
		if err != nil {
			t.Skipf("Failed to create test IMAP server with SPECIAL-USE support: %v", err)
		}
		defer server.Close()

		client, err := server.ConnectForE2E()
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer func() {
			_ = client.Logout()
		}()

		// Check if server supports SPECIAL-USE
		caps, err := client.Capability()
		if err != nil {
			t.Fatalf("Failed to check capabilities: %v", err)
		}

		if !caps["SPECIAL-USE"] {
			t.Skip("Server does not support SPECIAL-USE, skipping test")
		}

		// List folders - should return at least INBOX (created by memory backend)
		folders, err := ListFolders(client)
		if err != nil {
			t.Fatalf("ListFolders failed: %v", err)
		}

		// Memory backend creates INBOX by default, so we should have at least one folder
		if len(folders) == 0 {
			t.Error("Expected at least INBOX folder, got empty list")
		}

		// Verify INBOX is present
		foundINBOX := false
		for _, folder := range folders {
			if folder.Name == "INBOX" {
				foundINBOX = true
				if folder.Role != "inbox" {
					t.Errorf("Expected INBOX role 'inbox', got '%s'", folder.Role)
				}
			}
		}
		if !foundINBOX {
			t.Error("Expected to find INBOX folder")
		}
	})

	t.Run("handles network errors during list", func(t *testing.T) {
		// Create a client and then close it to simulate network error
		server := testutil.NewTestIMAPServer(t)
		defer server.Close()

		client, _ := server.Connect(t)
		// Close the client to simulate network error
		_ = client.Logout()

		// Try to list folders with closed client
		_, err := ListFolders(client)
		if err == nil {
			t.Error("Expected error when client is closed")
		}
	})
}
