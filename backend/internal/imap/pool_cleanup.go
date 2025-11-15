package imap

import (
	"time"
)

// startCleanupGoroutine runs a background goroutine that periodically cleans up idle connections.
// The goroutine will stop when cleanupCtx is cancelled (via Pool.Close()).
func (p *Pool) startCleanupGoroutine() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-p.cleanupCtx.Done():
				// Context cancelled - stop the ticker and exit
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
	for userID, pool := range p.workerPools {
		pool.mu.Lock()
		var toRemove []*clientWithMutex
		for _, conn := range pool.connections {
			if now.Sub(conn.GetLastUsed()) > workerIdleTimeout {
				toRemove = append(toRemove, conn)
			}
		}
		// Remove dead connections
		for _, conn := range toRemove {
			for i, c := range pool.connections {
				if c == conn {
					pool.connections = append(pool.connections[:i], pool.connections[i+1:]...)
					conn.Lock()
					_ = conn.GetClient().Logout()
					conn.Unlock()
					break
				}
			}
		}
		// Remove empty pools
		if len(pool.connections) == 0 {
			delete(p.workerPools, userID)
		}
		pool.mu.Unlock()
	}
}
