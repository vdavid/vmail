package imap

import (
	"fmt"
	"os"
	"testing"

	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestPool_ConcurrentAccess(t *testing.T) {
	// Set test mode to use non-TLS connections
	err := os.Setenv("VMAIL_TEST_MODE", "true")
	if err != nil {
		t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
	}
	defer func() {
		err := os.Unsetenv("VMAIL_TEST_MODE")
		if err != nil {
			t.Fatalf("Failed to unset VMAIL_TEST_MODE: %v", err)
		}
	}()

	server := testutil.NewTestIMAPServer(t)
	defer server.Close()

	pool := NewPool()
	defer pool.Close()

	t.Run("multiple goroutines creating clients simultaneously", func(t *testing.T) {
		const numGoroutines = 5
		const userID = "simultaneous-create-user"

		results := make(chan error, numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				err := pool.WithClient(userID, server.Address, server.Username(), server.Password(), func(client IMAPClient) error {
					// Client is automatically released when this function returns
					return nil
				})
				results <- err
			}()
		}

		// All should succeed without errors
		for i := 0; i < numGoroutines; i++ {
			if err := <-results; err != nil {
				t.Errorf("WithClient failed: %v", err)
			}
		}
	})

	t.Run("remove client while another goroutine is using it", func(t *testing.T) {
		const userID = "remove-while-using-user"

		// Use WithClient to get a client
		done := make(chan bool, 1)
		go func() {
			err := pool.WithClient(userID, server.Address, server.Username(), server.Password(), func(client IMAPClient) error {
				// Simulate using the client
				_ = client
				done <- true
				return nil
			})
			if err != nil {
				t.Errorf("WithClient failed: %v", err)
			}
		}()

		// Remove the client while it might be in use
		pool.RemoveClient(userID)

		// Wait for the goroutine to finish
		<-done
		// Should not panic
	})
}

func TestPool_EdgeCases(t *testing.T) {
	// Set test mode to use non-TLS connections
	err := os.Setenv("VMAIL_TEST_MODE", "true")
	if err != nil {
		t.Fatalf("Failed to set VMAIL_TEST_MODE: %v", err)
	}
	defer func() {
		err := os.Unsetenv("VMAIL_TEST_MODE")
		if err != nil {
			t.Fatalf("Failed to unset VMAIL_TEST_MODE: %v", err)
		}
	}()

	server := testutil.NewTestIMAPServer(t)
	defer server.Close()

	t.Run("pool with many users", func(t *testing.T) {
		pool := NewPool()
		defer pool.Close()

		const numUsers = 100
		for i := 0; i < numUsers; i++ {
			userID := fmt.Sprintf("user-%d", i)
			err := pool.WithClient(userID, server.Address, server.Username(), server.Password(), func(client IMAPClient) error {
				// Client is automatically released when this function returns
				return nil
			})
			if err != nil {
				t.Errorf("Failed to get client for user %s: %v", userID, err)
			}
		}

		// Verify all users have clients
		for i := 0; i < numUsers; i++ {
			userID := fmt.Sprintf("user-%d", i)
			pool.RemoveClient(userID)
			// Should not panic
		}
	})

	t.Run("close while clients are in use", func(t *testing.T) {
		pool := NewPool()

		// Use WithClient to get a client
		err := pool.WithClient("close-user", server.Address, server.Username(), server.Password(), func(client IMAPClient) error {
			// Client is automatically released when this function returns
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}

		// Close while the client might be in use
		pool.Close()

		// Should not panic
	})

	t.Run("remove client while in use", func(t *testing.T) {
		pool := NewPool()
		defer pool.Close()

		userID := "remove-in-use-user"
		err := pool.WithClient(userID, server.Address, server.Username(), server.Password(), func(client IMAPClient) error {
			// Client is automatically released when this function returns
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}

		// Remove while might be in use
		pool.RemoveClient(userID)
		// Should not panic
	})
}
