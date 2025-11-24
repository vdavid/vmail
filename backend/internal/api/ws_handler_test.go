package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
	"github.com/vdavid/vmail/backend/internal/testutil"
	"github.com/vdavid/vmail/backend/internal/testutil/mocks"
	ws "github.com/vdavid/vmail/backend/internal/websocket"
)

func TestWebSocketHandler_Connection(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	mockService := mocks.NewIMAPService(t)
	hub := ws.NewHub(10)
	handler := NewWebSocketHandler(pool, mockService, hub)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + server.URL[4:] + "?token=token"

	t.Run("connects successfully and stays open", func(t *testing.T) {
		// Expect StartIdleListener to be called and block
		startIdleCalled := make(chan struct{})
		mockService.On("StartIdleListener", mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				close(startIdleCalled)
				ctx := args.Get(0).(context.Context)
				<-ctx.Done()
			}).
			Return()
		// Also need to mock SyncThreadsForFolder which is called during connection setup
		mockService.On("SyncThreadsForFolder", mock.Anything, mock.Anything, mock.Anything).
			Return(nil)

		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("Expected status 101, got %d", resp.StatusCode)
		}

		t.Log("Connection established successfully")

		// Verify StartIdleListener was called
		select {
		case <-startIdleCalled:
			// success
		case <-time.After(2 * time.Second):
			t.Fatal("StartIdleListener was not called within timeout")
		}

		// Verify connection stays open for at least a bit
		// We'll read messages in a goroutine
		done := make(chan error, 1)
		go func() {
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					done <- err
					return
				}
			}
		}()

		select {
		case err := <-done:
			t.Errorf("Connection closed unexpectedly: %v", err)
		case <-time.After(1 * time.Second):
			// Connection is still open - good!
		}
	})

	t.Run("rejects connection without token", func(t *testing.T) {
		wsURLNoToken := "ws" + server.URL[4:]
		_, resp, err := websocket.DefaultDialer.Dial(wsURLNoToken, nil)
		if err == nil {
			t.Error("Expected connection to fail without token")
			if resp != nil {
				resp.Body.Close()
			}
		} else {
			t.Logf("Correctly rejected connection without token: %v", err)
		}
	})

	t.Run("rejects invalid token", func(t *testing.T) {
		// TODO: This test is skipped because ValidateToken is currently a stub that accepts all tokens.
		t.Skip("Token validation is not yet implemented - ValidateToken accepts all non-empty tokens")
	})
}
