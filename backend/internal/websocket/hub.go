package websocket

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client wraps a WebSocket connection.
type Client struct {
	conn *websocket.Conn
}

// Conn returns the underlying WebSocket connection.
func (c *Client) Conn() *websocket.Conn {
	return c.conn
}

// Hub manages active WebSocket connections per user.
// It supports multiple connections per user (e.g., multiple tabs).
type Hub struct {
	mu         sync.RWMutex
	clients    map[string]map[*Client]struct{} // userID -> set of clients
	maxPerUser int
}

// NewHub creates a new Hub with a per-user connection limit.
func NewHub(maxPerUser int) *Hub {
	if maxPerUser <= 0 {
		maxPerUser = 10
	}
	return &Hub{
		clients:    make(map[string]map[*Client]struct{}),
		maxPerUser: maxPerUser,
	}
}

// Register adds a WebSocket connection for the given user.
// If the per-user limit is exceeded, the new connection is closed and nil is returned.
func (h *Hub) Register(userID string, conn *websocket.Conn) *Client {
	h.mu.Lock()
	defer h.mu.Unlock()

	userClients, ok := h.clients[userID]
	if !ok {
		userClients = make(map[*Client]struct{})
		h.clients[userID] = userClients
	}

	if len(userClients) >= h.maxPerUser {
		log.Printf("websocket: user %s exceeded max connections (%d), closing new connection", userID, h.maxPerUser)
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections for this user"),
			// Use zero deadline - best effort.
			// See https://pkg.go.dev/github.com/gorilla/websocket#Conn.WriteControl
			// for details.
			//nolint:exhaustruct
			time.Time{},
		)
		_ = conn.Close()
		return nil
	}

	client := &Client{conn: conn}
	userClients[client] = struct{}{}
	return client
}

// Unregister removes a client for the given user and closes the connection.
func (h *Hub) Unregister(userID string, client *Client) {
	if client == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	userClients, ok := h.clients[userID]
	if !ok {
		_ = client.conn.Close()
		return
	}

	delete(userClients, client)

	if len(userClients) == 0 {
		delete(h.clients, userID)
	}

	_ = client.conn.Close()
}

// Send broadcasts a message to all active clients for the user.
func (h *Hub) Send(userID string, msg []byte) {
	h.mu.RLock()
	userClients := h.clients[userID]
	h.mu.RUnlock()

	if len(userClients) == 0 {
		return
	}

	for client := range userClients {
		if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("websocket: failed to write message for user %s: %v", userID, err)
			// Best-effort cleanup: unregister this client.
			go h.Unregister(userID, client)
		}
	}
}

// ActiveConnections returns the number of active WebSocket connections for a user.
func (h *Hub) ActiveConnections(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients[userID])
}
