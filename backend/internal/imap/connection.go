package imap

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/emersion/go-imap/client"
)

// connectionRole indicates the purpose of a connection.
type connectionRole int

const (
	// roleWorker indicates a worker connection. There can be multiple worker connections per user.
	roleWorker connectionRole = iota
	// roleListener indicates a listener connection. There can be only one listener connection per user.
	roleListener
)

// clientWithMutex wraps an IMAP client with a mutex for thread-safe access.
// Each connection has its own mutex to allow concurrent access to different connections
// while serializing access to the same connection.
type clientWithMutex struct {
	client   *client.Client
	mu       sync.Mutex
	lastUsed time.Time
	role     connectionRole
}

// Lock acquires the mutex for thread-safe access to the underlying client.
func (c *clientWithMutex) Lock() {
	c.mu.Lock()
}

// Unlock releases the mutex.
func (c *clientWithMutex) Unlock() {
	c.mu.Unlock()
}

// GetClient returns the underlying IMAP client (for internal use).
// Caller must hold the lock before calling this.
func (c *clientWithMutex) GetClient() *client.Client {
	return c.client
}

// UpdateLastUsed updates the lastUsed timestamp to now.
func (c *clientWithMutex) UpdateLastUsed() {
	c.lastUsed = time.Now()
}

// GetLastUsed returns the lastUsed timestamp.
func (c *clientWithMutex) GetLastUsed() time.Time {
	return c.lastUsed
}

// GetRole returns the connection role (worker or listener).
func (c *clientWithMutex) GetRole() connectionRole {
	return c.role
}

// ConnectToIMAP connects to the IMAP server with a 5-second timeout.
// useTLS: true for production (TLS), false for tests (non-TLS).
func ConnectToIMAP(server string, useTLS bool) (*client.Client, error) {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	if useTLS {
		c, err := client.DialWithDialerTLS(dialer, server, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to dial with TLS: %w", err)
		}
		return c, nil
	}

	// Non-TLS connection for testing
	c, err := client.DialWithDialer(dialer, server)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	return c, nil
}

// Login authenticates with the IMAP server.
func Login(c *client.Client, username, password string) error {
	if err := c.Login(username, password); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	return nil
}
