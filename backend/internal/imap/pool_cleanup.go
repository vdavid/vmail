package imap

import (
	"time"
)

// startCleanupGoroutine runs a background goroutine that periodically cleans up idle connections.
// The goroutine will stop when cleanupCtx is canceled (via Pool.Close()).
func (p *Pool) startCleanupGoroutine() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-p.cleanupCtx.Done():
				// Context canceled - stop the ticker and exit
				return
			case <-ticker.C:
				// Periodic cleanup
				p.cleanupIdleConnections()
			}
		}
	}()
}

// cleanupIdleConnections removes worker connections that have been idle too long.
func (p *Pool) cleanupIdleConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for userID, set := range p.workerSets {
		set.mu.Lock()
		var toRemove []*threadSafeClient
		for _, client := range set.clients {
			if now.Sub(client.GetLastUsed()) > workerIdleTimeout {
				toRemove = append(toRemove, client)
			}
		}
		// Remove idle clients
		for _, client := range toRemove {
			for i, c := range set.clients {
				if c == client {
					set.clients = append(set.clients[:i], set.clients[i+1:]...)
					client.Lock()
					_ = client.GetClient().Logout()
					client.Unlock()
					break
				}
			}
		}
		// Remove empty sets
		if len(set.clients) == 0 {
			delete(p.workerSets, userID)
		}
		set.mu.Unlock()
	}
}
