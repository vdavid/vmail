package imap

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Pool manages IMAP connections per user.
// Each user has at most one connection, which is reused across requests.
// FIXME-ARCHITECTURE: IMAP clients from go-imap are NOT thread-safe.
// Multiple goroutines using the same client concurrently can cause race conditions.
// The current design assumes one request per user at a time, but this is not enforced.
// Consider:
// 1. Adding a per-client mutex to serialize access to each client
// 2. Using a connection pool per user (multiple connections per user)
// 3. Documenting that concurrent requests for the same user are not supported
type Pool struct {
	clients map[string]*client.Client
	mu      sync.RWMutex
}

// NewPool creates a new IMAP connection pool.
func NewPool() *Pool {
	return &Pool{
		clients: make(map[string]*client.Client),
	}
}

// getClientConcrete gets or creates an IMAP client for a user (internal use).
// Returns the concrete *client.Client type for internal operations.
// FIXME-SMELL: Race condition between checking state and removing client.
// If client state is checked, found to be dead, but another goroutine is using it,
// we could remove it while it's still in use. Consider double-checking after acquiring
// write lock, or using a more robust connection health check.
// FIXME-ARCHITECTURE: No connection timeout or idle timeout - connections stay open indefinitely.
// Consider adding:
// 1. Idle timeout (close connections after X minutes of inactivity)
// 2. Connection health checks (ping/NOOP command)
// 3. Maximum pool size to prevent unbounded growth
func (p *Pool) getClientConcrete(userID, server, username, password string) (*client.Client, error) {
	p.mu.RLock()
	c, exists := p.clients[userID]
	p.mu.RUnlock()

	if exists && c != nil {
		// Check if the connection is still alive
		state := c.State()
		// ConnState values: 0=NotAuthenticated, 1=Authenticated, 2=Selected
		if state == imap.AuthenticatedState || state == imap.SelectedState {
			return c, nil
		}
		// Connection is dead, remove it
		// FIXME-SMELL: Double-check after acquiring write lock to avoid race condition.
		// Another goroutine might have already removed it or recreated it.
		p.mu.Lock()
		// Double-check: client might have been removed or recreated by another goroutine
		if p.clients[userID] == c {
			delete(p.clients, userID)
		}
		p.mu.Unlock()
	}

	// Create a new connection (use TLS in production, non-TLS for tests)
	// Check environment variable for test mode
	useTLS := os.Getenv("VMAIL_TEST_MODE") != "true"
	c, err := ConnectToIMAP(server, useTLS)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	if err := Login(c, username, password); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	// FIXME-SMELL: Another goroutine might have created a client for this user
	// between when we checked and now. We should check again and close the old one if it exists.
	p.mu.Lock()
	if existingClient, exists := p.clients[userID]; exists && existingClient != c {
		// Another goroutine created a client - close ours and use the existing one
		_ = c.Logout()
		c = existingClient
	} else {
		p.clients[userID] = c
	}
	p.mu.Unlock()

	return c, nil
}

// GetClient gets or creates an IMAP client for a user.
// Implements IMAPPool interface - returns IMAPClient for testability.
func (p *Pool) GetClient(userID, server, username, password string) (IMAPClient, error) {
	c, err := p.getClientConcrete(userID, server, username, password)
	if err != nil {
		return nil, err
	}
	return &ClientWrapper{client: c}, nil
}

// RemoveClient removes a client from the pool and logs out.
func (p *Pool) RemoveClient(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	c, exists := p.clients[userID]
	if exists {
		_ = c.Logout()
		delete(p.clients, userID)
	}
}

// Close closes all connections in the pool.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for userID, c := range p.clients {
		if err := c.Logout(); err != nil {
			log.Printf("Failed to logout IMAP client for user %s: %v", userID, err)
		}
		delete(p.clients, userID)
	}
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
