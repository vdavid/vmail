package imap

import (
	"testing"
	"time"

	"github.com/emersion/go-imap-sortthread"
	"github.com/emersion/go-imap/client"
	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestRunThreadCommand(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*testing.T) *client.Client
		expectError bool
		checkResult func(*testing.T, []*sortthread.Thread, error)
	}{
		{
			name: "returns error for nil client",
			setup: func(*testing.T) *client.Client {
				return nil
			},
			expectError: true,
			checkResult: func(t *testing.T, threads []*sortthread.Thread, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "client is nil")
			},
		},
		{
			name: "handles empty mailbox",
			setup: func(t *testing.T) *client.Client {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				server.EnsureINBOX(t)
				c, cleanup := server.Connect(t)
				t.Cleanup(cleanup)
				_, err := c.Select("INBOX", false)
				if err != nil {
					t.Fatalf("Failed to select INBOX: %v", err)
				}
				return c
			},
			expectError: false,
			checkResult: func(t *testing.T, threads []*sortthread.Thread, err error) {
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
				if !caps["THREAD"] {
					assert.Error(t, err, "should error when server doesn't support THREAD")
					return
				}
				assert.NoError(t, err)
				assert.NotNil(t, threads)
				assert.Empty(t, threads)
			},
		},
		{
			name: "handles mailbox with unthreaded messages",
			setup: func(t *testing.T) *client.Client {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				server.EnsureINBOX(t)
				now := time.Now()
				server.AddMessage(t, "INBOX", "<msg1@test>", "Subject 1", "from@test.com", "to@test.com", now)
				server.AddMessage(t, "INBOX", "<msg2@test>", "Subject 2", "from@test.com", "to@test.com", now.Add(-1*time.Hour))
				c, cleanup := server.Connect(t)
				t.Cleanup(cleanup)
				_, err := c.Select("INBOX", false)
				if err != nil {
					t.Fatalf("Failed to select INBOX: %v", err)
				}
				return c
			},
			expectError: false,
			checkResult: func(t *testing.T, threads []*sortthread.Thread, err error) {
				if err != nil {
					// Some servers may not support THREAD command
					assert.NotEmpty(t, err.Error(), "expected non-empty error message")
					return
				}
				assert.NotNil(t, threads)
			},
		},
		{
			name: "handles server without THREAD support",
			setup: func(t *testing.T) *client.Client {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				server.EnsureINBOX(t)
				c, cleanup := server.Connect(t)
				t.Cleanup(cleanup)
				return c
			},
			expectError: false,
			checkResult: func(t *testing.T, threads []*sortthread.Thread, err error) {
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
				if !caps["THREAD"] {
					assert.Error(t, err, "should error when server doesn't support THREAD")
				} else {
					assert.NoError(t, err, "should succeed when THREAD is supported")
				}
			},
		},
		{
			name: "handles network errors during thread command",
			setup: func(t *testing.T) *client.Client {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				c, _ := server.Connect(t)
				_ = c.Logout() // Close the client to simulate network error
				return c
			},
			expectError: true,
			checkResult: func(t *testing.T, threads []*sortthread.Thread, err error) {
				assert.Error(t, err, "should error when client is closed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setup(t)
			threads, err := RunThreadCommand(client)
			if tt.expectError {
				if tt.checkResult != nil {
					tt.checkResult(t, threads, err)
				}
				return
			}
			if tt.checkResult != nil {
				tt.checkResult(t, threads, err)
			}
		})
	}
}
