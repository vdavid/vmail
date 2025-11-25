package imap

import (
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestFetchMessageHeaders(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*testing.T) (*client.Client, []uint32)
		expectError bool
		checkResult func(*testing.T, []*imap.Message, error)
	}{
		{
			name: "returns error for nil client",
			setup: func(*testing.T) (*client.Client, []uint32) {
				return nil, []uint32{1, 2, 3}
			},
			expectError: true,
			checkResult: func(t *testing.T, messages []*imap.Message, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "client is nil")
			},
		},
		{
			name: "returns empty slice for empty UIDs",
			setup: func(t *testing.T) (*client.Client, []uint32) {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				c, cleanup := server.Connect(t)
				t.Cleanup(cleanup)
				return c, []uint32{}
			},
			expectError: false,
			checkResult: func(t *testing.T, messages []*imap.Message, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, messages)
				assert.Empty(t, messages)
			},
		},
		{
			name: "fetches message headers successfully",
			setup: func(t *testing.T) (*client.Client, []uint32) {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				server.EnsureINBOX(t)
				uid := server.AddMessage(t, "INBOX", "<test@example.com>", "Test Subject", "from@example.com", "to@example.com", time.Now())
				c, cleanup := server.Connect(t)
				t.Cleanup(cleanup)
				_, err := c.Select("INBOX", false)
				if err != nil {
					t.Fatalf("Failed to select INBOX: %v", err)
				}
				return c, []uint32{uid}
			},
			expectError: false,
			checkResult: func(t *testing.T, messages []*imap.Message, err error) {
				assert.NoError(t, err)
				assert.Len(t, messages, 1)
				if len(messages) > 0 {
					assert.NotNil(t, messages[0].Envelope)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, uids := tt.setup(t)
			messages, err := FetchMessageHeaders(client, uids)
			if tt.expectError {
				if tt.checkResult != nil {
					tt.checkResult(t, messages, err)
				}
				return
			}
			if tt.checkResult != nil {
				tt.checkResult(t, messages, err)
			}
		})
	}
}

func TestFetchFullMessage(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*testing.T) (*client.Client, uint32)
		expectError bool
		checkResult func(*testing.T, *imap.Message, error)
	}{
		{
			name: "returns error for nil client",
			setup: func(*testing.T) (*client.Client, uint32) {
				return nil, 1
			},
			expectError: true,
			checkResult: func(t *testing.T, msg *imap.Message, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "client is nil")
			},
		},
		{
			name: "fetches full message successfully",
			setup: func(t *testing.T) (*client.Client, uint32) {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				server.EnsureINBOX(t)
				uid := server.AddMessage(t, "INBOX", "<test@example.com>", "Test Subject", "from@example.com", "to@example.com", time.Now())
				c, cleanup := server.Connect(t)
				t.Cleanup(cleanup)
				_, err := c.Select("INBOX", false)
				if err != nil {
					t.Fatalf("Failed to select INBOX: %v", err)
				}
				return c, uid
			},
			expectError: false,
			checkResult: func(t *testing.T, msg *imap.Message, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, msg)
				assert.NotNil(t, msg.Envelope)
			},
		},
		{
			name: "handles message without body structure",
			setup: func(t *testing.T) (*client.Client, uint32) {
				server := testutil.NewTestIMAPServer(t)
				t.Cleanup(server.Close)
				server.EnsureINBOX(t)
				uid := server.AddMessage(t, "INBOX", "<test@example.com>", "Test Subject", "from@example.com", "to@example.com", time.Now())
				c, cleanup := server.Connect(t)
				t.Cleanup(cleanup)
				_, err := c.Select("INBOX", false)
				if err != nil {
					t.Fatalf("Failed to select INBOX: %v", err)
				}
				return c, uid
			},
			expectError: false,
			checkResult: func(t *testing.T, msg *imap.Message, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, msg, "message should be returned even if body structure is nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, uid := tt.setup(t)
			msg, err := FetchFullMessage(client, uid)
			if tt.expectError {
				if tt.checkResult != nil {
					tt.checkResult(t, msg, err)
				}
				return
			}
			if tt.checkResult != nil {
				tt.checkResult(t, msg, err)
			}
		})
	}
}
