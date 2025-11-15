package imap

import (
	"fmt"
	"os"
	"time"

	"github.com/emersion/go-imap"
)

// GetListenerConnection gets or creates a listener connection for a user.
// Listener connections are dedicated connections for IDLE command.
// Returns a locked connection that must be unlocked by the caller.
// Thread-safe: uses double-check locking pattern.
func (p *Pool) GetListenerConnection(userID, server, username, password string) (*clientWithMutex, error) {
	// First check without lock
	p.mu.RLock()
	listener, exists := p.listeners[userID]
	p.mu.RUnlock()

	if exists {
		listener.Lock()
		// Double-check after acquiring lock
		p.mu.RLock()
		existingListener, stillExists := p.listeners[userID]
		p.mu.RUnlock()

		if stillExists && existingListener == listener {
			// Check if connection is healthy
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
			// Retry with new connection
			return p.GetListenerConnection(userID, server, username, password)
		}
	}

	// Need to create new listener connection
	useTLS := os.Getenv("VMAIL_TEST_MODE") != "true"
	c, err := ConnectToIMAP(server, useTLS)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	if err := Login(c, username, password); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	// Wrap in clientWithMutex
	listener = &clientWithMutex{
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
