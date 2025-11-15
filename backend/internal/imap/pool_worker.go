package imap

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// userWorkerPool manages multiple worker connections for a single user.
// Uses a semaphore to limit concurrent connections (max 3 by default).
type userWorkerPool struct {
	connections []*clientWithMutex
	semaphore   chan struct{} // Limits concurrent connections (max 3)
	mu          sync.Mutex
}

// acquire gets a connection from the pool, blocking if at max capacity.
// Returns the connection (locked) and a release function that must be called when done.
// If no connection is available, returns nil and the caller should create a new one.
func (p *userWorkerPool) acquire() (*clientWithMutex, func()) {
	// Block until a slot is available
	p.semaphore <- struct{}{}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Find an available connection (not in use)
	for _, conn := range p.connections {
		// Connection is available if we can acquire its lock immediately
		if conn.mu.TryLock() {
			conn.UpdateLastUsed()
			// Keep it locked - caller will unlock when done
			return conn, func() {
				conn.Unlock()
				<-p.semaphore // Release semaphore slot
			}
		}
	}

	// No available connection - caller will need to create one
	<-p.semaphore         // Release semaphore slot temporarily
	return nil, func() {} // No-op release function
}

// addConnection adds a new connection to the pool.
func (p *userWorkerPool) addConnection(conn *clientWithMutex) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connections = append(p.connections, conn)
}

// close closes all connections in the pool.
func (p *userWorkerPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		conn.Lock()
		if err := conn.client.Logout(); err != nil {
			log.Printf("Failed to logout worker connection: %v", err)
		}
		conn.Unlock()
	}
	p.connections = nil
}

// getOrCreateWorkerPool gets or creates a worker pool for a user.
// Thread-safe: uses double-check locking pattern.
func (p *Pool) getOrCreateWorkerPool(userID string) *userWorkerPool {
	// First check without lock
	p.mu.RLock()
	pool, exists := p.workerPools[userID]
	p.mu.RUnlock()

	if exists {
		return pool
	}

	// Need to create - acquire write lock
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check: another goroutine might have created it
	if pool, exists := p.workerPools[userID]; exists {
		return pool
	}

	// Create new pool
	pool = &userWorkerPool{
		connections: make([]*clientWithMutex, 0),
		semaphore:   make(chan struct{}, p.maxWorkers),
	}
	p.workerPools[userID] = pool
	return pool
}

// getWorkerConnection gets or creates a worker connection for a user.
// Returns a locked connection and a release function that must be called when done.
// Thread-safe: uses double-check locking and proper synchronization.
func (p *Pool) getWorkerConnection(userID, server, username, password string) (*clientWithMutex, func(), error) {
	pool := p.getOrCreateWorkerPool(userID)

	// Try to acquire an existing connection
	conn, release := pool.acquire()
	if conn != nil {
		// Connection is already locked from acquire()
		// Check if connection is healthy
		state := conn.GetClient().State()
		if state == imap.AuthenticatedState || state == imap.SelectedState {
			// Check if we need health check
			lastUsed := conn.GetLastUsed()
			if time.Since(lastUsed) > healthCheckThreshold {
				if !p.checkConnectionHealth(conn) {
					// Connection is dead, unlock and remove it
					conn.Unlock()
					release()
					// Remove from pool and create new one
					p.removeDeadConnection(pool, conn)
					// Fall through to create new connection
				} else {
					// Connection is healthy, update timestamp
					conn.UpdateLastUsed()
					return conn, release, nil // Caller must call release() when done
				}
			} else {
				// Connection is healthy and recently used
				conn.UpdateLastUsed()
				return conn, release, nil // Caller must call release() when done
			}
		} else {
			// Connection is dead
			conn.Unlock()
			release()
			p.removeDeadConnection(pool, conn)
			// Fall through to create new connection
		}
	}

	// Need to create new connection
	// Acquire semaphore slot
	pool.semaphore <- struct{}{}

	// Use a flag to track if we should release in defer
	// We'll manually release on error paths, so defer should not release in those cases
	shouldReleaseInDefer := true
	defer func() {
		if shouldReleaseInDefer {
			<-pool.semaphore
		}
	}()

	// Double-check: another goroutine might have created a connection while we were waiting
	pool.mu.Lock()
	for _, existingConn := range pool.connections {
		if existingConn.mu.TryLock() {
			state := existingConn.GetClient().State()
			if state == imap.AuthenticatedState || state == imap.SelectedState {
				existingConn.UpdateLastUsed()
				pool.mu.Unlock()
				// Return with release function
				// Don't release in defer since we're returning a connection
				shouldReleaseInDefer = false
				release := func() {
					existingConn.Unlock()
					<-pool.semaphore
				}
				return existingConn, release, nil // Caller must call release() when done
			}
			existingConn.mu.Unlock()
		}
	}
	pool.mu.Unlock()

	// Create new connection
	useTLS := os.Getenv("VMAIL_TEST_MODE") != "true"
	c, err := ConnectToIMAP(server, useTLS)
	if err != nil {
		shouldReleaseInDefer = false // Don't release in defer, we'll do it manually
		<-pool.semaphore             // Release semaphore on error
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}

	if err := Login(c, username, password); err != nil {
		shouldReleaseInDefer = false // Don't release in defer, we'll do it manually
		_ = c.Logout()
		<-pool.semaphore // Release semaphore on error
		return nil, nil, fmt.Errorf("failed to login: %w", err)
	}

	// Wrap in clientWithMutex
	newConn := &clientWithMutex{
		client:   c,
		lastUsed: time.Now(),
		role:     roleWorker,
	}
	conn = newConn

	// Add to pool
	pool.addConnection(conn)
	conn.Lock() // Lock before returning

	// Don't release in defer - the release function will handle it
	shouldReleaseInDefer = false
	// Create release function for the new connection
	newRelease := func() {
		conn.Unlock()
		<-pool.semaphore
	}
	return conn, newRelease, nil
}

// removeDeadConnection removes a dead connection from the pool.
func (p *Pool) removeDeadConnection(pool *userWorkerPool, conn *clientWithMutex) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	for i, c := range pool.connections {
		if c == conn {
			// Remove from slice
			pool.connections = append(pool.connections[:i], pool.connections[i+1:]...)
			// Close connection
			conn.Lock()
			_ = conn.client.Logout()
			conn.Unlock()
			break
		}
	}
}

// checkConnectionHealth performs a NOOP command to check if connection is alive.
// The connection must be locked before calling this.
func (p *Pool) checkConnectionHealth(conn *clientWithMutex) bool {
	// Connection is already locked by caller
	if err := conn.client.Noop(); err != nil {
		return false
	}
	return true
}

// getClientConcrete gets or creates a worker connection for a user (internal use).
// Returns the concrete *client.Client type for internal operations.
// Thread-safe: The connection is locked during the operation. For short-lived operations
// (like Select, Fetch), this is acceptable. The connection will be automatically unlocked
// after a short delay to allow reuse. For long-running operations, consider using getWorkerConnection
// directly for better control.
//
// Note: This method uses a goroutine to automatically release the connection after 5 seconds.
// This is a workaround for backward compatibility. In the future, callers should be refactored
// to use getWorkerConnection directly and manage the release themselves.
func (p *Pool) getClientConcrete(userID, server, username, password string) (*client.Client, error) {
	conn, release, err := p.getWorkerConnection(userID, server, username, password)
	if err != nil {
		return nil, err
	}
	// For backward compatibility, we unlock after a short delay
	// This allows the connection to be reused while still providing thread safety
	// during the immediate operation. Most operations (Select, Fetch) complete in < 1 second.
	// Using 5 seconds instead of 30 to avoid holding connections too long.
	go func() {
		time.Sleep(5 * time.Second)
		// Check if pool is still open before releasing
		// If the pool is closed, don't try to release (would cause panic or deadlock)
		select {
		case <-p.cleanupCtx.Done():
			// Pool is closed, don't try to release
			return
		default:
			release()
		}
	}()
	return conn.GetClient(), nil
}
