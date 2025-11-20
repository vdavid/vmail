package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
	ws "github.com/vdavid/vmail/backend/internal/websocket"
)

func TestWebSocketHandler_Connection(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	// Create a mock IMAP service
	mockIMAP := &mockIMAPServiceForWS{
		startIdleListenerCalled: false,
		startIdleListenerCtx:    make(chan context.Context, 1),
	}

	hub := ws.NewHub(10)
	handler := NewWebSocketHandler(pool, mockIMAP, hub)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + server.URL[4:] + "?token=token"

	t.Run("connects successfully and stays open", func(t *testing.T) {
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer func(conn *websocket.Conn) {
			err := conn.Close()
			if err != nil {
				t.Errorf("Failed to close connection: %v", err)
			}
		}(conn)

		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("Expected status 101, got %d", resp.StatusCode)
		}

		t.Log("Connection established successfully")

		// Verify connection stays open for at least 3-4 seconds
		// We'll read messages in a goroutine and check if the connection closes prematurely
		done := make(chan error, 1)
		messageReceived := make(chan bool, 1)

		go func() {
			for {
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					done <- err
					return
				}
				t.Logf("Received message: type=%d, content=%s", messageType, string(message))
				select {
				case messageReceived <- true:
				default:
				}
			}
		}()

		// Wait a number of seconds and check if the connection is still open
		startTime := time.Now()
		select {
		case err := <-done:
			duration := time.Since(startTime)
			t.Errorf("Connection closed unexpectedly after %v: %v", duration, err)
		case <-messageReceived:
			t.Log("Received a message (connection is active)")
			// Continue waiting to see if it stays open
			select {
			case err := <-done:
				duration := time.Since(startTime)
				t.Errorf("Connection closed after receiving message (after %v): %v", duration, err)
			case <-time.After(4 * time.Second):
				duration := time.Since(startTime)
				t.Logf("Connection stayed open for %v after message", duration)
			}
		case <-time.After(5 * time.Second):
			// Connection is still open - good!
			duration := time.Since(startTime)
			t.Logf("Connection stayed open for %v - SUCCESS", duration)
		}

		// Check if IDLE listener was started
		if !mockIMAP.startIdleListenerCalled {
			t.Error("Expected StartIdleListener to be called")
		}
	})

	t.Run("rejects connection without token", func(t *testing.T) {
		wsURLNoToken := "ws" + server.URL[4:]
		_, resp, err := websocket.DefaultDialer.Dial(wsURLNoToken, nil)
		if err == nil {
			t.Error("Expected connection to fail without token")
			if resp != nil {
				err := resp.Body.Close()
				if err != nil {
					return
				}
			}
		} else {
			t.Logf("Correctly rejected connection without token: %v", err)
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		// TODO: This test is skipped because ValidateToken is currently a stub that accepts all tokens.
		// Once proper token validation is implemented, this test should verify rejection of invalid tokens.
		t.Skip("Token validation is not yet implemented - ValidateToken accepts all non-empty tokens")
	})
}

// mockIMAPServiceForWS is a mock implementation of IMAPService for WebSocket tests
type mockIMAPServiceForWS struct {
	startIdleListenerCalled bool
	startIdleListenerCtx    chan context.Context
}

func (m *mockIMAPServiceForWS) StartIdleListener(ctx context.Context, _ string, _ *ws.Hub) {
	m.startIdleListenerCalled = true
	select {
	case m.startIdleListenerCtx <- ctx:
	default:
	}
	// Block until context is cancelled (simulating IDLE)
	<-ctx.Done()
}

// Implement other required IMAPService methods (return nil/empty for now)
func (m *mockIMAPServiceForWS) SyncThreadsForFolder(context.Context, string, string) error {
	return nil
}

func (m *mockIMAPServiceForWS) ShouldSyncFolder(context.Context, string, string) (bool, error) {
	return false, nil
}

func (m *mockIMAPServiceForWS) SyncFullMessage(context.Context, string, string, int64) error {
	return nil
}

func (m *mockIMAPServiceForWS) SyncFullMessages(context.Context, string, []imap.MessageToSync) error {
	return nil
}

func (m *mockIMAPServiceForWS) Search(context.Context, string, string, int, int) ([]*models.Thread, int, error) {
	return nil, 0, nil
}

func (m *mockIMAPServiceForWS) Close() {}
