package imap

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	// workerIdleTimeout is the maximum time a worker connection can be idle before being closed.
	workerIdleTimeout = 10 * time.Minute
	// healthCheckThreshold is the idle time after which we perform a health check before reuse.
	healthCheckThreshold = 1 * time.Minute
)

// Pool manages IMAP connections per user.
// Supports two types of connections:
// - Worker connections: 1-3 connections per user for API handlers (SEARCH, FETCH, STORE)
// - Listener connections: 1 dedicated connection per user for IDLE command
//
// Thread safety: Each connection is wrapped with a mutex to ensure thread-safe access.
// Multiple goroutines can use different connections concurrently, but access to the same
// connection is serialized.
type Pool struct {
	workerSets    map[string]*workerClientSet  // userID -> worker client set
	listeners     map[string]*threadSafeClient // userID -> listener connection
	mu            sync.RWMutex
	maxWorkers    int // Maximum worker connections per user (default: 3)
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
}

// NewPool creates a new IMAP connection pool with the default worker limit.
func NewPool() *Pool {
	return NewPoolWithMaxWorkers(3)
}

// NewPoolWithMaxWorkers creates a new IMAP connection pool with a configurable
// maximum number of worker connections per user.
func NewPoolWithMaxWorkers(maxWorkers int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		workerSets:    make(map[string]*workerClientSet),
		listeners:     make(map[string]*threadSafeClient),
		maxWorkers:    maxWorkers,
		cleanupCtx:    ctx,
		cleanupCancel: cancel,
	}
	go p.startCleanupGoroutine()
	return p
}

// GetClient gets or creates an IMAP client for a user.
// Implements IMAPPool interface - returns IMAPClient and a release function
// that must be called when the caller is done with the client.
func (p *Pool) GetClient(userID, server, username, password string) (IMAPClient, func(), error) {
	tsClient, release, err := p.getWorkerConnection(userID, server, username, password)
	if err != nil {
		return nil, nil, err
	}
	return &ClientWrapper{client: tsClient.GetClient()}, release, nil
}

// RemoveClient removes all connections (worker and listener) for a user from the pool.
func (p *Pool) RemoveClient(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove worker set
	if set, exists := p.workerSets[userID]; exists {
		set.close()
		delete(p.workerSets, userID)
	}

	// Remove listener
	if listener, exists := p.listeners[userID]; exists {
		listener.Lock()
		_ = listener.GetClient().Logout()
		listener.Unlock()
		delete(p.listeners, userID)
	}
}

// Close closes all connections in the pool and stops the cleanup goroutine.
func (p *Pool) Close() {
	// Stop cleanup goroutine
	p.cleanupCancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Close all worker sets
	for userID, set := range p.workerSets {
		set.close()
		delete(p.workerSets, userID)
	}

	// Close all listener connections
	for userID, listener := range p.listeners {
		// Try to lock - if we can't, the listener is in use
		if listener.TryLock() {
			if err := listener.GetClient().Logout(); err != nil {
				log.Printf("Failed to logout listener connection for user %s: %v", userID, err)
			}
			listener.Unlock()
		} else {
			// Listener is locked (in use) - try to close anyway during shutdown
			_ = listener.GetClient().Logout()
		}
		delete(p.listeners, userID)
	}
}
