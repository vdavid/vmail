package imap

import (
	"github.com/emersion/go-imap/client"
	"github.com/vdavid/vmail/backend/internal/models"
)

// IMAPClient defines the interface for IMAP client operations needed by handlers.
// This allows handlers to be tested with mock implementations.
// Note: The stutter in the naming is intentional because go-imap already has a client.Client.
//
//goland:noinspection GoNameStartsWithPackageName
type IMAPClient interface {
	// ListFolders lists all folders on the IMAP server with their roles determined by SPECIAL-USE attributes.
	ListFolders() ([]*models.Folder, error)
}

// IMAPPool defines the interface for the IMAP connection pool.
// This allows handlers to be tested with mock implementations.
// Note: The stutter in the naming is intentional because we have a struct called Pool.
//
//goland:noinspection GoNameStartsWithPackageName
type IMAPPool interface {
	// WithClient gets an IMAP client for a user and calls the provided function with it.
	// The client is automatically released when the function returns, ensuring worker slots
	// are freed promptly. This is the safe way to use the pool - it's impossible to forget
	// to release the client.
	WithClient(userID, server, username, password string, fn func(IMAPClient) error) error

	// RemoveClient removes a client from the pool (useful when a connection is broken).
	RemoveClient(userID string)

	// Close closes all connections in the pool.
	Close()
}

// ClientWrapper wraps a go-imap client.Client to implement IMAPClient interface.
type ClientWrapper struct {
	client *client.Client
}

// ListFolders lists all folders on the IMAP server with their roles determined by SPECIAL-USE attributes.
func (w *ClientWrapper) ListFolders() ([]*models.Folder, error) {
	return ListFolders(w.client)
}

// ListenerClient defines the interface for listener client operations.
// This allows the IDLE feature to work with the thread-safe wrapper
// without exposing implementation details.
type ListenerClient interface {
	// Lock acquires the mutex for thread-safe access to the underlying client.
	Lock()
	// Unlock releases the mutex.
	Unlock()
	// GetClient returns the underlying IMAP client.
	// Caller must hold the lock before calling this.
	GetClient() *client.Client
}

// Ensure Pool implements IMAPPool interface
var _ IMAPPool = (*Pool)(nil)

// Ensure threadSafeClient implements ListenerClient interface
var _ ListenerClient = (*threadSafeClient)(nil)
