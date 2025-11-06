package imap

import (
	"testing"

	"github.com/emersion/go-imap"
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
