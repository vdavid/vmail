package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	ws "github.com/vdavid/vmail/backend/internal/websocket"
)

// WebSocketHandler handles the /api/v1/ws endpoint for real-time updates.
type WebSocketHandler struct {
	pool *pgxpool.Pool
	imap interface {
		imap.IMAPService
		StartIdleListener(ctx context.Context, userID string, hub *ws.Hub)
	}
	hub         *ws.Hub
	mu          sync.Mutex
	idleCancels map[string]context.CancelFunc
}

// NewWebSocketHandler creates a new WebSocketHandler instance.
func NewWebSocketHandler(pool *pgxpool.Pool, imapService imap.IMAPService, hub *ws.Hub) *WebSocketHandler {
	return &WebSocketHandler{
		pool:        pool,
		imap:        imapService,
		hub:         hub,
		idleCancels: make(map[string]context.CancelFunc),
	}
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// For now, allow all origins. This server is expected to be used
		// behind a reverse proxy in a trusted environment.
		return true
	},
}

// Handle upgrades the HTTP connection to a WebSocket and registers it with the Hub.
// Authentication is handled via query parameter (?token=...) since WebSocket connections
// cannot set custom headers in browsers. The token is validated using the same ValidateToken
// function used by the RequireAuth middleware.
func (h *WebSocketHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract token from query parameter (WebSocket connections can't set headers).
	token := r.URL.Query().Get("token")
	if token == "" {
		// Fallback to Authorization header if query parameter is not provided.
		// This allows testing with tools that can set headers.
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			fields := strings.Fields(authHeader)
			if len(fields) >= 2 && strings.EqualFold(fields[0], "Bearer") {
				token = strings.TrimSpace(strings.Join(fields[1:], " "))
			}
		}
	}

	if token == "" {
		log.Printf("WebSocketHandler: No token provided (neither query parameter nor Authorization header)")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate token using the same function as RequireAuth middleware.
	userEmail, err := auth.ValidateToken(token)
	if err != nil {
		log.Printf("WebSocketHandler: Token validation failed: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get or create user from email.
	userID, err := db.GetOrCreateUser(ctx, h.pool, userEmail)
	if err != nil {
		log.Printf("WebSocketHandler: Failed to get/create user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocketHandler: failed to upgrade connection for user %s: %v", userID, err)
		return
	}

	// Check if this is the first connection for this user (before registering).
	// This determines if we need to catch up on missed emails.
	isFirstConnection := h.hub.ActiveConnections(userID) == 0

	client := h.hub.Register(userID, conn)
	if client == nil {
		log.Printf("WebSocketHandler: Connection rejected for user %s (max connections exceeded)", userID)
		return
	}

	// Temporary logging
	log.Printf("WebSocketHandler: WebSocket connection established for user %s", userID)

	// Ensure an IDLE listener is running for this user.
	h.ensureIdleListener(userID)

	// If this is the first connection, immediately sync INBOX to catch up on missed emails.
	// This ensures emails that arrived while no WebSocket was connected are synced.
	if isFirstConnection {
		// Temporary logging
		log.Printf("WebSocketHandler: First connection for user %s, triggering immediate INBOX sync", userID)
		go func() {
			// Use background context to avoid cancellation if request context is cancelled.
			// The sync should complete even if the WebSocket connection is established.
			syncCtx := context.Background()
			if err := h.imap.SyncThreadsForFolder(syncCtx, userID, "INBOX"); err != nil {
				log.Printf("WebSocketHandler: Failed to sync INBOX for user %s on connection: %v", userID, err)
			} else {
				// Temporary logging
				log.Printf("WebSocketHandler: Successfully synced INBOX for user %s on connection", userID)
			}
		}()
	}

	// Read loop to keep the connection open and detect disconnects.
	go h.readLoop(userID, client)
}

// ensureIdleListener starts an IMAP IDLE listener for the user if one is not already running.
func (h *WebSocketHandler) ensureIdleListener(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.idleCancels[userID]; exists {
		return
	}

	// Temporary logging
	log.Printf("WebSocketHandler: Starting IDLE listener for user %s", userID)
	idleCtx, cancel := context.WithCancel(context.Background())
	h.idleCancels[userID] = cancel

	go func(ctx context.Context, uid string, cancelFn context.CancelFunc) {
		h.imap.StartIdleListener(ctx, uid, h.hub)

		// When the listener exits, clean up the cancel function if it's still registered.
		h.mu.Lock()
		delete(h.idleCancels, uid)
		h.mu.Unlock()
	}(idleCtx, userID, cancel)
}

// readLoop reads messages from the WebSocket until the connection is closed.
// When the connection closes, it unregisters the client and may stop the IDLE listener
// if there are no more active connections for the user.
func (h *WebSocketHandler) readLoop(userID string, client *ws.Client) {
	conn := client.Conn()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	// Unregister the client and close the connection.
	h.hub.Unregister(userID, client)

	// If there are no active connections left for this user, stop the IDLE listener.
	if h.hub.ActiveConnections(userID) == 0 {
		// Temporary logging
		log.Printf("WebSocketHandler: No active connections remaining for user %s, stopping IDLE listener", userID)
		h.mu.Lock()
		if cancel, exists := h.idleCancels[userID]; exists {
			cancel()
			delete(h.idleCancels, userID)
		}
		h.mu.Unlock()
	}
}
