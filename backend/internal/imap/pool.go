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
	workerPools   map[string]*userWorkerPool  // userID -> worker pool
	listeners     map[string]*clientWithMutex // userID -> listener connection
	mu            sync.RWMutex
	maxWorkers    int // Maximum worker connections per user (default: 3)
	cleanupCtx    context.Context
	cleanupCancel context.CancelFunc
}

// NewPool creates a new IMAP connection pool.
func NewPool() *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		workerPools:   make(map[string]*userWorkerPool),
		listeners:     make(map[string]*clientWithMutex),
		maxWorkers:    3,
		cleanupCtx:    ctx,
		cleanupCancel: cancel,
	}
	go p.startCleanupGoroutine()
	return p
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

// RemoveClient removes all connections (worker and listener) for a user from the pool.
func (p *Pool) RemoveClient(userID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove worker pool
	if pool, exists := p.workerPools[userID]; exists {
		pool.close()
		delete(p.workerPools, userID)
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

	// Close all worker pools
	for userID, pool := range p.workerPools {
		pool.close()
		delete(p.workerPools, userID)
	}

	// Close all listener connections
	for userID, listener := range p.listeners {
		listener.Lock()
		if err := listener.GetClient().Logout(); err != nil {
			log.Printf("Failed to logout listener connection for user %s: %v", userID, err)
		}
		listener.Unlock()
		delete(p.listeners, userID)
	}
}
