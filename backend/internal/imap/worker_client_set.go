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
func (s *workerClientSet) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, client := range s.clients {
		client.Lock()
		if err := client.client.Logout(); err != nil {
			log.Printf("Failed to logout worker client: %v", err)
		}
		client.Unlock()
	}
	s.clients = nil
}
