package imap

import (
	"testing"

	"github.com/emersion/go-imap/client"
	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestListFolders(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*testing.T) *client.Client
		expectError bool
		checkResult func(*testing.T, []*models.Folder, error)
	}{
		{
			name: "returns error for nil client",
			setup: func(*testing.T) *client.Client {
				return nil
			},
			expectError: true,
			checkResult: func(t *testing.T, folders []*models.Folder, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "client is nil")
			},
		},
		{
			name: "returns error for server without SPECIAL-USE support",
			setup: func(t *testing.T) *client.Client {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				c, cleanup := server.Connect(t)
				t.Cleanup(cleanup)
				return c
			},
			expectError: false,
			checkResult: func(t *testing.T, folders []*models.Folder, err error) {
				// Check capabilities to determine expected behavior
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				c, cleanup := server.Connect(t)
				t.Cleanup(cleanup)
				defer cleanup()
				caps, capErr := c.Capability()
				if capErr != nil {
					t.Fatalf("Failed to check capabilities: %v", capErr)
				}
				if !caps["SPECIAL-USE"] {
					assert.Error(t, err, "should error when server doesn't support SPECIAL-USE")
					if err != nil {
						assert.NotEmpty(t, err.Error(), "expected non-empty error message")
					}
				} else {
					assert.NoError(t, err, "should succeed when SPECIAL-USE is supported")
					assert.NotNil(t, folders)
				}
			},
		},
		{
			name: "handles empty folder list",
			setup: func(t *testing.T) *client.Client {
				server, err := testutil.NewTestIMAPServerForE2E()
				if err != nil {
					t.Skipf("Failed to create test IMAP server with SPECIAL-USE support: %v", err)
				}
				t.Cleanup(server.Close)
				c, err := server.ConnectForE2E()
				if err != nil {
					t.Fatalf("Failed to connect: %v", err)
				}
				t.Cleanup(func() {
					_ = c.Logout()
				})
				return c
			},
			expectError: false,
			checkResult: func(t *testing.T, folders []*models.Folder, err error) {
				// Check capabilities
				server, serverErr := testutil.NewTestIMAPServerForE2E()
				if serverErr != nil {
					t.Skipf("Failed to create test IMAP server: %v", serverErr)
				}
				defer server.Close()
				c, connErr := server.ConnectForE2E()
				if connErr != nil {
					t.Fatalf("Failed to connect: %v", connErr)
				}
				defer func() {
					_ = c.Logout()
				}()
				caps, capErr := c.Capability()
				if capErr != nil {
					t.Fatalf("Failed to check capabilities: %v", capErr)
				}
				if !caps["SPECIAL-USE"] {
					t.Skip("Server does not support SPECIAL-USE, skipping test")
				}
				assert.NoError(t, err)
				assert.NotEmpty(t, folders, "should have at least INBOX folder")
				foundINBOX := false
				for _, folder := range folders {
					if folder.Name == "INBOX" {
						foundINBOX = true
						assert.Equal(t, "inbox", folder.Role)
					}
				}
				assert.True(t, foundINBOX, "should find INBOX folder")
			},
		},
		{
			name: "handles network errors during list",
			setup: func(t *testing.T) *client.Client {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				c, _ := server.Connect(t)
				_ = c.Logout() // Close the client to simulate network error
				return c
			},
			expectError: true,
			checkResult: func(t *testing.T, folders []*models.Folder, err error) {
				assert.Error(t, err, "should error when client is closed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setup(t)
			folders, err := ListFolders(client)
			if tt.expectError {
				if tt.checkResult != nil {
					tt.checkResult(t, folders, err)
				}
				return
			}
			if tt.checkResult != nil {
				tt.checkResult(t, folders, err)
			}
		})
	}
}
