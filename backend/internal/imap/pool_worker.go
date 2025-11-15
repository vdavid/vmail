package imap

import (
	"fmt"
	"os"
	"time"

	"github.com/emersion/go-imap"
)

// getOrCreateWorkerSet gets or creates a worker client set for a user.
// Thread-safe: uses double-check locking pattern.
func (p *Pool) getOrCreateWorkerSet(userID string) *workerClientSet {
	// First check without lock
	p.mu.RLock()
	set, exists := p.workerSets[userID]
	p.mu.RUnlock()

	if exists {
		return set
	}

	// Need to create - acquire write lock
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check: another goroutine might have created it
	if set, exists := p.workerSets[userID]; exists {
		return set
	}

	// Create new set
	set = &workerClientSet{
		clients:   make([]*threadSafeClient, 0),
		semaphore: make(chan struct{}, p.maxWorkers),
	}
	p.workerSets[userID] = set
	return set
}

// getWorkerConnection gets or creates a worker client for a user.
// Returns a locked client and a release function that must be called when done.
// Thread-safe: uses double-check locking and proper synchronization.
func (p *Pool) getWorkerConnection(userID, server, username, password string) (*threadSafeClient, func(), error) {
	set := p.getOrCreateWorkerSet(userID)

	// Try to acquire an existing client
	tsClient, release := set.acquire()
	if tsClient != nil {
		// Client is already locked from acquire()
		// Check if client is healthy
		state := tsClient.GetClient().State()
		if state == imap.AuthenticatedState || state == imap.SelectedState {
			// Check if we need a health check
			lastUsed := tsClient.GetLastUsed()
			if time.Since(lastUsed) > healthCheckThreshold {
				if !p.checkConnectionHealth(tsClient) {
					// Client is dead, unlock and remove it
					tsClient.Unlock()
					release()
					// Remove from the set and create a new one
					p.removeDeadClient(set, tsClient)
					// Fall through to create a new client
				} else {
					// Client is healthy, update timestamp
					tsClient.UpdateLastUsed()
					return tsClient, release, nil // Caller must call release() when done
				}
			} else {
				// Client is healthy and recently used
				tsClient.UpdateLastUsed()
				return tsClient, release, nil // Caller must call release() when done
			}
		} else {
			// Client is dead
			tsClient.Unlock()
			release()
			p.removeDeadClient(set, tsClient)
			// Fall through to create a new client
		}
	}

	// Need to create a new client
	// Acquire semaphore slot
	set.semaphore <- struct{}{}

	// Use a flag to track if we should release in defer
	// We'll manually release on error paths, so defer should not release in those cases
	shouldReleaseInDefer := true
	defer func() {
		if shouldReleaseInDefer {
			<-set.semaphore
		}
	}()

	// Double-check: another goroutine might have created a client while we were waiting
	set.mu.Lock()
	for _, existingClient := range set.clients {
		if existingClient.mu.TryLock() {
			state := existingClient.GetClient().State()
			if state == imap.AuthenticatedState || state == imap.SelectedState {
				existingClient.UpdateLastUsed()
				set.mu.Unlock()
				// Return with release function
				// Don't release in defer since we're returning a client
				shouldReleaseInDefer = false
				release := func() {
					existingClient.Unlock()
					<-set.semaphore
				}
				return existingClient, release, nil // Caller must call release() when done
			}
			existingClient.mu.Unlock()
		}
	}
	set.mu.Unlock()

	// Create new client
	useTLS := os.Getenv("VMAIL_TEST_MODE") != "true"
	c, err := ConnectToIMAP(server, useTLS)
	if err != nil {
		shouldReleaseInDefer = false // Don't release in defer, we'll do it manually
		<-set.semaphore              // Release semaphore on error
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}

	if err := Login(c, username, password); err != nil {
		shouldReleaseInDefer = false // Don't release in defer, we'll do it manually
		_ = c.Logout()
		<-set.semaphore // Release semaphore on error
		return nil, nil, fmt.Errorf("failed to login: %w", err)
	}

	// Wrap in threadSafeClient
	newClient := &threadSafeClient{
		client:   c,
		lastUsed: time.Now(),
		role:     roleWorker,
	}
	tsClient = newClient

	// Add to set
	set.addClient(tsClient)
	tsClient.Lock() // Lock before returning

	// Don't release in defer - the release function will handle it
	shouldReleaseInDefer = false
	// Create release function for the new client
	newRelease := func() {
		tsClient.Unlock()
		<-set.semaphore
	}
	return tsClient, newRelease, nil
}

// removeDeadClient removes a dead client from the set.
func (p *Pool) removeDeadClient(set *workerClientSet, client *threadSafeClient) {
	set.mu.Lock()
	defer set.mu.Unlock()

	for i, c := range set.clients {
		if c == client {
			// Remove from slice
			set.clients = append(set.clients[:i], set.clients[i+1:]...)
			// Close client
			client.Lock()
			_ = client.client.Logout()
			client.Unlock()
			break
		}
	}
}

// checkConnectionHealth performs a NOOP command to check if client is alive.
// The client must be locked before calling this.
func (p *Pool) checkConnectionHealth(client *threadSafeClient) bool {
	// The caller has already locked the client
	if err := client.client.Noop(); err != nil {
		return false
	}
	return true
}
