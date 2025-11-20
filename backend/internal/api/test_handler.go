package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	imapinternal "github.com/vdavid/vmail/backend/internal/imap"
	ws "github.com/vdavid/vmail/backend/internal/websocket"
)

// TestHandler provides test-only endpoints used by E2E tests.
// These endpoints are only registered in test environments.
type TestHandler struct {
	pool        *pgxpool.Pool
	encryptor   *crypto.Encryptor
	imapService imapinternal.IMAPService
	hub         *ws.Hub
}

// NewTestHandler creates a new TestHandler instance.
func NewTestHandler(pool *pgxpool.Pool, encryptor *crypto.Encryptor, imapService imapinternal.IMAPService, hub *ws.Hub) *TestHandler {
	return &TestHandler{
		pool:        pool,
		encryptor:   encryptor,
		imapService: imapService,
		hub:         hub,
	}
}

type addIMAPMessageRequest struct {
	Folder  string `json:"folder"`
	Subject string `json:"subject"`
	From    string `json:"from"`
	To      string `json:"to"`
}

// AddIMAPMessage appends a test message to the user's IMAP folder.
// It is used by E2E tests to simulate new incoming mail.
func (h *TestHandler) AddIMAPMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx, w, h.pool)
	if !ok {
		return
	}

	req, err := h.parseRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client, err := h.connectToIMAP(ctx, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = client.Logout()
	}()

	if err := h.appendMessage(client, req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.syncAndNotify(ctx, userID, req.Folder)

	w.WriteHeader(http.StatusNoContent)
}

// parseRequest parses and validates the request body.
func (h *TestHandler) parseRequest(r *http.Request) (*addIMAPMessageRequest, error) {
	var req addIMAPMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, fmt.Errorf("invalid JSON body")
	}

	if req.Folder == "" {
		req.Folder = "INBOX"
	}

	if req.Subject == "" || req.From == "" || req.To == "" {
		return nil, fmt.Errorf("subject, from, and to are required")
	}

	return &req, nil
}

// connectToIMAP connects to the IMAP server and selects the folder.
func (h *TestHandler) connectToIMAP(ctx context.Context, userID string) (*imapclient.Client, error) {
	settings, err := db.GetUserSettings(ctx, h.pool, userID)
	if err != nil {
		log.Printf("TestHandler: failed to get user settings: %v", err)
		return nil, fmt.Errorf("failed to get user settings")
	}

	imapPassword, err := h.encryptor.Decrypt(settings.EncryptedIMAPPassword)
	if err != nil {
		log.Printf("TestHandler: failed to decrypt IMAP password: %v", err)
		return nil, fmt.Errorf("failed to decrypt IMAP password")
	}

	useTLS := os.Getenv("VMAIL_TEST_MODE") != "true"

	client, err := imapinternal.ConnectToIMAP(settings.IMAPServerHostname, useTLS)
	if err != nil {
		log.Printf("TestHandler: failed to connect to IMAP server: %v", err)
		return nil, fmt.Errorf("failed to connect to IMAP server")
	}

	if err := imapinternal.Login(client, settings.IMAPUsername, imapPassword); err != nil {
		log.Printf("TestHandler: failed to login to IMAP server: %v", err)
		_ = client.Logout()
		return nil, fmt.Errorf("failed to login to IMAP server")
	}

	return client, nil
}

// appendMessage appends a message to the specified IMAP folder.
func (h *TestHandler) appendMessage(client *imapclient.Client, req *addIMAPMessageRequest) error {
	// Select the folder.
	if _, err := client.Select(req.Folder, false); err != nil {
		log.Printf("TestHandler: failed to select folder %s: %v", req.Folder, err)
		return fmt.Errorf("failed to select IMAP folder")
	}

	// Construct a simple RFC 822 message.
	messageID := fmt.Sprintf("<e2e-%d@vmail.local>", time.Now().UnixNano())
	now := time.Now()
	messageBody := fmt.Sprintf(`Message-ID: %s
Date: %s
From: %s
To: %s
Subject: %s
Content-Type: text/plain; charset=utf-8

E2E test message.
`, messageID, now.Format(time.RFC1123Z), req.From, req.To, req.Subject)

	flags := []string{imap.SeenFlag}
	if err := client.Append(req.Folder, flags, now, strings.NewReader(messageBody)); err != nil {
		log.Printf("TestHandler: failed to append message: %v", err)
		return fmt.Errorf("failed to append message to IMAP folder")
	}

	return nil
}

// syncAndNotify syncs the folder and sends a WebSocket notification.
func (h *TestHandler) syncAndNotify(ctx context.Context, userID, folder string) {
	if err := h.imapService.SyncThreadsForFolder(ctx, userID, folder); err != nil {
		log.Printf("TestHandler: failed to sync folder %s for user %s: %v", folder, userID, err)
		return
	}

	msg := struct {
		Type   string `json:"type"`
		Folder string `json:"folder"`
	}{
		Type:   "new_email",
		Folder: folder,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("TestHandler: failed to marshal new_email message: %v", err)
		return
	}

	h.hub.Send(userID, payload)
}
