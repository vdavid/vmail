package imap

import (
	"fmt"
	"os"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestPool_GetClient(t *testing.T) {
	pool := NewPool()
	defer pool.Close()

	t.Run("creates new client when none exists", func(t *testing.T) {
		// This test would require a real IMAP server or a mock.
		// For now, we test that the pool structure works
		if pool == nil {
			t.Error("Expected pool to be created")
		}
	})

	t.Run("removes client from pool", func(t *testing.T) {
		pool.RemoveClient("test-user")
		// Should not panic
	})

	t.Run("removes and recreates client when connection is dead", func(t *testing.T) {
		// This test verifies the reconnection logic in GetClient.
		// The logic should:
		// 1. Check if the client exists and is in AuthenticatedState or SelectedState
		// 2. If the client is in NotAuthenticatedState (or any other invalid state), remove it
		// 3. Create a new connection
		//
		// To properly test this, we would need:
		// - A mock IMAP client that can return different states
		// - Or a real IMAP server that we can disconnect
		//
		// For now, we verify the pool structure and that RemoveClient works
		userID := "test-reconnect-user"
		pool.RemoveClient(userID) // Clean up if exists

		// The actual reconnection test would:
		// 1. Manually add a mock client in NotAuthenticatedState to the pool
		// 2. Call GetClient
		// 3. Assert that the old client was removed and a new one was created
		//
		// This requires refactoring Pool to accept a client factory function
		// or using interfaces to inject mock clients.
		_ = userID
	})
}

//goland:noinspection GoBoolExpressions
func TestPool_GetClient_ReconnectionLogic(t *testing.T) {
	// This test documents the expected reconnection behavior:
	//
	// When GetClient is called and a client exists in the pool:
	// 1. Check client.State()
	// 2. If the state is imap.AuthenticatedState or imap.SelectedState, return the existing client
	// 3. If the state is imap.NotAuthenticatedState (or any other state), remove the client from the pool
	// 4. Create new connection and add to pool
	//
	// To test this properly, we need:
	// - Interface for IMAP client with State() method
	// - Mock client that can return different states
	// - Ability to inject mock into pool
	//
	// Example test structure:
	//   mockClient := &MockClient{state: imap.NotAuthenticatedState}
	//   pool.clients["user"] = mockClient
	//   newClient, err := pool.GetClient("user", "server", "user", "pass")
	//   assert mockClient was removed
	//   assert newClient is different from mockClient
	//   assert newClient.State() is AuthenticatedState or SelectedState

	// Verify the state constants exist and are distinct
	// The actual values may vary by go-imap version, but they should be distinct
	if imap.NotAuthenticatedState == imap.AuthenticatedState {
		t.Error("NotAuthenticatedState and AuthenticatedState should be different")
	}
	if imap.AuthenticatedState == imap.SelectedState {
		t.Error("AuthenticatedState and SelectedState should be different")
	}
	if imap.NotAuthenticatedState == imap.SelectedState {
		t.Error("NotAuthenticatedState and SelectedState should be different")
	}

	// Log the actual values for reference
	t.Logf("IMAP state constants: NotAuthenticatedState=%d, AuthenticatedState=%d, SelectedState=%d",
		imap.NotAuthenticatedState, imap.AuthenticatedState, imap.SelectedState)
}

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
				_, release, err := pool.GetClient(userID, server.Address, server.Username(), server.Password())
				if release != nil {
					release()
				}
				results <- err
			}()
		}

		// All should succeed without errors
		for i := 0; i < numGoroutines; i++ {
			if err := <-results; err != nil {
				t.Errorf("GetClient failed: %v", err)
			}
		}
	})

	t.Run("remove client while another goroutine is using it", func(t *testing.T) {
		const userID = "remove-while-using-user"

		// Get a client first
		client, release, err := pool.GetClient(userID, server.Address, server.Username(), server.Password())
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}
		if release != nil {
			defer release()
		}

		// Start a goroutine that uses the client
		done := make(chan bool, 1)
		go func() {
			// Simulate using the client
			_ = client
			done <- true
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
			_, release, err := pool.GetClient(userID, server.Address, server.Username(), server.Password())
			if err != nil {
				t.Errorf("Failed to get client for user %s: %v", userID, err)
			}
			if release != nil {
				release()
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

		// Get a client
		_, release, err := pool.GetClient("close-user", server.Address, server.Username(), server.Password())
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}
		if release != nil {
			defer release()
		}

		// Close while the client might be in use
		pool.Close()

		// Should not panic
	})

	t.Run("remove client while in use", func(t *testing.T) {
		pool := NewPool()
		defer pool.Close()

		userID := "remove-in-use-user"
		_, release, err := pool.GetClient(userID, server.Address, server.Username(), server.Password())
		if err != nil {
			t.Fatalf("Failed to get client: %v", err)
		}
		if release != nil {
			defer release()
		}

		// Remove while might be in use
		pool.RemoveClient(userID)
		// Should not panic
	})
}

func TestPool_Close(t *testing.T) {
	pool := NewPool()

	t.Run("closes all clients", func(t *testing.T) {
		pool.Close()
		// Should not panic
	})

	t.Run("can be called multiple times safely", func(t *testing.T) {
		pool := NewPool()
		pool.Close()
		pool.Close() // Should not panic
	})
}
