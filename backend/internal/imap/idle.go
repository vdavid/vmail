package imap

import (
	"context"
	"encoding/json"
	"log"
	"time"

	idle "github.com/emersion/go-imap-idle"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/vdavid/vmail/backend/internal/websocket"
)

// idleListenerSleep is the backoff duration after an error before retrying IDLE.
const idleListenerSleep = 10 * time.Second

// StartIdleListener runs an IMAP IDLE loop for a user and pushes new email events to the Hub.
// It listens on the INBOX folder only.
// This function blocks until the context is canceled.
func (s *Service) StartIdleListener(ctx context.Context, userID string, hub *websocket.Hub) {
	for {
		// Exit when context is canceled.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// If the user has no active WebSocket connections, avoid doing work.
		if hub.ActiveConnections(userID) == 0 {
			time.Sleep(idleListenerSleep)
			continue
		}

		listener, err := s.getListenerConnection(ctx, userID)
		if err != nil {
			time.Sleep(idleListenerSleep)
			continue
		}

		// Ensure we always unlock the listener.
		func() {
			defer listener.Unlock()
			s.runIdleLoop(ctx, userID, listener.GetClient(), hub)
		}()

		// Small backoff before trying again.
		time.Sleep(idleListenerSleep)
	}
}

// getListenerConnection gets settings and establishes a listener connection.
func (s *Service) getListenerConnection(ctx context.Context, userID string) (ListenerClient, error) {
	settings, imapPassword, err := s.getSettingsAndPassword(ctx, userID)
	if err != nil {
		log.Printf("IMAP IDLE: failed to get settings for user %s: %v", userID, err)
		return nil, err
	}

	listener, err := s.imapPool.GetListenerConnection(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword)
	if err != nil {
		log.Printf("IMAP IDLE: failed to get listener connection for user %s: %v", userID, err)
		return nil, err
	}

	return listener, nil
}

// runIdleLoop runs the IDLE command and handles mailbox updates.
func (s *Service) runIdleLoop(ctx context.Context, userID string, client *imapclient.Client, hub *websocket.Hub) {
	// Select INBOX for IDLE.
	if _, err := client.Select("INBOX", false); err != nil {
		log.Printf("IMAP IDLE: failed to select INBOX for user %s: %v", userID, err)
		s.imapPool.RemoveListenerConnection(userID)
		return
	}

	idleClient := idle.NewClient(client)

	// Create a channel to receive mailbox updates.
	updates := make(chan imapclient.Update, 10)
	client.Updates = updates

	// Start IDLE in a goroutine so we can listen for updates.
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- idleClient.IdleWithFallback(stop, 5*time.Second)
	}()

	for {
		select {
		case <-ctx.Done():
			// Stop idling and return.
			close(stop)
			return
		case err := <-done:
			if err != nil {
				log.Printf("IMAP IDLE: idle loop ended with error for user %s: %v", userID, err)
				s.imapPool.RemoveListenerConnection(userID)
			}
			return
		case update := <-updates:
			if update == nil {
				continue
			}
			s.handleMailboxUpdate(ctx, userID, update, hub)
		}
	}
}

// handleMailboxUpdate processes a mailbox update and syncs/notifies if needed.
func (s *Service) handleMailboxUpdate(ctx context.Context, userID string, update imapclient.Update, hub *websocket.Hub) {
	// MailboxUpdate updates can indicate new messages.
	mboxUpdate, ok := update.(*imapclient.MailboxUpdate)
	if !ok || mboxUpdate.Mailbox == nil {
		return
	}

	status := mboxUpdate.Mailbox
	if status.Name != "INBOX" || status.Messages == 0 {
		return
	}

	// Perform incremental sync for INBOX immediately.
	if err := s.SyncThreadsForFolder(ctx, userID, "INBOX"); err != nil {
		log.Printf("IMAP IDLE: failed to sync INBOX for user %s: %v", userID, err)
		return
	}

	// Notify frontend via WebSocket.
	s.sendNewEmailNotification(userID, hub)
}

// sendNewEmailNotification sends a WebSocket notification about new email.
func (s *Service) sendNewEmailNotification(userID string, hub *websocket.Hub) {
	msg := struct {
		Type   string `json:"type"`
		Folder string `json:"folder"`
	}{
		Type:   "new_email",
		Folder: "INBOX",
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("IMAP IDLE: failed to marshal new_email message: %v", err)
		return
	}
	hub.Send(userID, payload)
}
