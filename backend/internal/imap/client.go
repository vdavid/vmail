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
		p.mu.Lock()
		delete(p.clients, userID)
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

	p.mu.Lock()
	p.clients[userID] = c
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
