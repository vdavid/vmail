package imap

import (
	"github.com/emersion/go-imap/client"
)

// IMAPClient defines the interface for IMAP client operations needed by handlers.
// This allows handlers to be tested with mock implementations.
// Note: The stutter in the naming is intentional because go-imap already has a client.Client.
//
//goland:noinspection GoNameStartsWithPackageName
type IMAPClient interface {
	// ListFolders lists all folders on the IMAP server.
	ListFolders() ([]string, error)
}

// IMAPPool defines the interface for the IMAP connection pool.
// This allows handlers to be tested with mock implementations.
// Note: The stutter in the naming is intentional because we have a struct called Pool.
//
//goland:noinspection GoNameStartsWithPackageName
type IMAPPool interface {
	// GetClient gets or creates an IMAP client for a user.
	GetClient(userID, server, username, password string) (IMAPClient, error)

	// Close closes all connections in the pool.
	Close()
}

// ClientWrapper wraps a go-imap client.Client to implement IMAPClient interface.
type ClientWrapper struct {
	client *client.Client
}

// ListFolders lists all folders on the IMAP server.
func (w *ClientWrapper) ListFolders() ([]string, error) {
	return ListFolders(w.client)
}

// Ensure Pool implements IMAPPool interface
var _ IMAPPool = (*Pool)(nil)
