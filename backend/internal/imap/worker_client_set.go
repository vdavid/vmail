package imap

import (
	"log"
	"sync"
)

// workerClientSet manages multiple worker clients for a single user.
// Uses a semaphore to limit concurrent connections (max 3 by default).
type workerClientSet struct {
	clients   []*threadSafeClient
	semaphore chan struct{} // Limits concurrent connections (max 3)
	mu        sync.Mutex
}

// acquire gets a client from the set, blocking if at max capacity.
// Returns the client (locked) and a release function that must be called when done.
// If no client is available, returns nil and the caller should create a new one.
func (s *workerClientSet) acquire() (*threadSafeClient, func()) {
	// Block until a slot is available
	s.semaphore <- struct{}{}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find an available client (not in use)
	for _, client := range s.clients {
		// Client is available if we can acquire its lock immediately
		if client.mu.TryLock() {
			client.UpdateLastUsed()
			// Keep it locked - caller will unlock when done
			return client, func() {
				client.Unlock()
				<-s.semaphore // Release semaphore slot
			}
		}
	}

	// No available client - caller will need to create one
	<-s.semaphore         // Release semaphore slot temporarily
	return nil, func() {} // No-op release function
}

// addClient adds a new client to the set.
func (s *workerClientSet) addClient(client *threadSafeClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients = append(s.clients, client)
}

// close closes all clients in the set.
// If a client is currently locked (in use), it will be skipped.
// The auto-release goroutine will handle closing it when it sees the pool is closed.
func (s *workerClientSet) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, client := range s.clients {
		// Try to lock - if we can't, the client is in use and will be closed
		// by the auto-release goroutine when it sees cleanupCtx.Done()
		if client.TryLock() {
			if err := client.client.Logout(); err != nil {
				log.Printf("Failed to logout worker client: %v", err)
			}
			client.Unlock()
		} else {
			// Client is locked (in use) - skip it
			// The auto-release goroutine will see cleanupCtx.Done() and won't release,
			// but we should still try to close the underlying connection
			// Note: This is not thread-safe, but we're shutting down so it's acceptable
			_ = client.client.Logout()
		}
	}
	s.clients = nil
}
