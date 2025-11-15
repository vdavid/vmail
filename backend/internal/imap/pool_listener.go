package imap

import (
	"fmt"
	"os"
	"time"

	"github.com/emersion/go-imap"
)

// getListenerConnection gets or creates a listener client for a user.
// Listener clients are dedicated clients for IDLE command.
// Returns a locked client that must be unlocked by the caller.
// Thread-safe: uses double-check locking pattern.
func (p *Pool) getListenerConnection(userID, server, username, password string) (*threadSafeClient, error) {
	// First check without a lock
	p.mu.RLock()
	listener, exists := p.listeners[userID]
	p.mu.RUnlock()

	if exists {
		listener.Lock()
		// Double-check after acquiring a lock
		p.mu.RLock()
		existingListener, stillExists := p.listeners[userID]
		p.mu.RUnlock()

		if stillExists && existingListener == listener {
			// Check if the connection is healthy
			state := listener.GetClient().State()
			if state == imap.AuthenticatedState || state == imap.SelectedState {
				return listener, nil // Caller must unlock
			}
			// Connection is dead, unlock and remove it
			listener.Unlock()
			p.mu.Lock()
			if p.listeners[userID] == listener {
				delete(p.listeners, userID)
			}
			p.mu.Unlock()
			// Close dead connection
			_ = listener.GetClient().Logout()
		} else {
			// Another goroutine removed/recreated it
			listener.Unlock()
			// Retry with a new connection
			return p.getListenerConnection(userID, server, username, password)
		}
	}

	// Need to create a new listener connection
	useTLS := os.Getenv("VMAIL_TEST_MODE") != "true"
	c, err := ConnectToIMAP(server, useTLS)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	if err := Login(c, username, password); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	// Wrap in threadSafeClient
	listener = &threadSafeClient{
		client:   c,
		lastUsed: time.Now(),
		role:     roleListener,
	}

	// Double-check before adding
	p.mu.Lock()
	if existingListener, exists := p.listeners[userID]; exists {
		// Another goroutine created it - close ours and use existing
		_ = c.Logout()
		p.mu.Unlock()
		listener = existingListener
		listener.Lock()
		return listener, nil
	}
	p.listeners[userID] = listener
	p.mu.Unlock()

	listener.Lock() // Lock before returning
	return listener, nil
}

// RemoveListenerConnection removes a listener connection from the pool.
func (p *Pool) RemoveListenerConnection(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	listener, exists := p.listeners[userID]
	if exists {
		listener.Lock()
		_ = listener.GetClient().Logout()
		listener.Unlock()
		delete(p.listeners, userID)
	}
}
