package imap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/emersion/go-imap"
	sortthread "github.com/emersion/go-imap-sortthread"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
)

// Service handles IMAP operations and caching.
type Service struct {
	pool       *pgxpool.Pool
	clientPool *Pool
	encryptor  *crypto.Encryptor
	cacheTTL   time.Duration
}

// NewService creates a new IMAP service.
func NewService(pool *pgxpool.Pool, encryptor *crypto.Encryptor) *Service {
	return &Service{
		pool:       pool,
		clientPool: NewPool(),
		encryptor:  encryptor,
		cacheTTL:   5 * time.Minute, // Default cache TTL
	}
}

// getSettingsAndPassword gets user settings and decrypts the IMAP password.
func (s *Service) getSettingsAndPassword(ctx context.Context, userID string) (*models.UserSettings, string, error) {
	settings, err := db.GetUserSettings(ctx, s.pool, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user settings: %w", err)
	}

	imapPassword, err := s.encryptor.Decrypt(settings.EncryptedIMAPPassword)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decrypt IMAP password: %w", err)
	}

	return settings, imapPassword, nil
}

// getClientAndSelectFolder gets user settings, decrypts the password, gets the IMAP client, and selects the folder.
// Returns the client and mailbox status, or an error.
func (s *Service) getClientAndSelectFolder(ctx context.Context, userID, folderName string) (*imapclient.Client, *imap.MailboxStatus, error) {
	settings, imapPassword, err := s.getSettingsAndPassword(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	// Get IMAP client (internal use - need concrete type)
	client, err := s.clientPool.getClientConcrete(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get IMAP client: %w", err)
	}

	// Select the folder
	mbox, err := client.Select(folderName, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to select folder %s: %w", folderName, err)
	}

	return client, mbox, nil
}

// SyncThreadsForFolder syncs threads from IMAP for a specific folder.
func (s *Service) SyncThreadsForFolder(ctx context.Context, userID, folderName string) error {
	client, mbox, err := s.getClientAndSelectFolder(ctx, userID, folderName)
	if err != nil {
		return err
	}

	log.Printf("Selected folder %s: %d messages", folderName, mbox.Messages)

	// Run THREAD command
	threads, err := RunThreadCommand(client)
	if err != nil {
		return fmt.Errorf("failed to run THREAD command: %w", err)
	}

	log.Printf("Found %d threads in folder %s", len(threads), folderName)

	// Collect all UIDs from threads (threads are recursive)
	allUIDs := make([]uint32, 0)
	uidToThreadRootMap := make(map[uint32]uint32) // Map message UID to root thread UID

	// Recursively map all messages in a thread to their root UID
	var mapThreadToRoot func(*sortthread.Thread, uint32)
	mapThreadToRoot = func(thread *sortthread.Thread, rootUID uint32) {
		if thread == nil {
			return
		}
		// Map this message to the root
		uidToThreadRootMap[thread.Id] = rootUID
		allUIDs = append(allUIDs, thread.Id)
		// Recursively process all children
		for _, child := range thread.Children {
			mapThreadToRoot(child, rootUID)
		}
	}

	// Build maps: find root for each top-level thread
	rootUIDs := make([]uint32, 0)
	for _, thread := range threads {
		if thread == nil {
			continue
		}
		// The root UID is the thread's own ID (top-level threads are roots)
		rootUID := thread.Id
		rootUIDs = append(rootUIDs, rootUID)
		// Map all messages in this thread tree to this root
		mapThreadToRoot(thread, rootUID)
	}

	if len(allUIDs) == 0 {
		log.Printf("No messages found in folder %s", folderName)
		return nil
	}

	// Fetch message headers for all messages
	messages, err := FetchMessageHeaders(client, allUIDs)
	if err != nil {
		return fmt.Errorf("failed to fetch message headers: %w", err)
	}

	// Build map of UID to message for a quick lookup
	uidToMessageMap := make(map[uint32]*imap.Message)
	for _, msg := range messages {
		uidToMessageMap[msg.Uid] = msg
	}

	// Build map of root UID to stable thread ID (Message-ID)
	// We already have this data from the uidToMessageMap. No new fetch needed.
	rootUIDToStableID := make(map[uint32]string)
	for _, rootUID := range rootUIDs {
		if rootMsg, found := uidToMessageMap[rootUID]; found {
			if rootMsg.Envelope != nil && len(rootMsg.Envelope.MessageId) > 0 {
				rootUIDToStableID[rootUID] = rootMsg.Envelope.MessageId
			}
		}
	}

	log.Printf("Fetched %d message headers", len(messages))

	// Process each message
	for _, imapMsg := range messages {
		rootUID, ok := uidToThreadRootMap[imapMsg.Uid]
		if !ok {
			log.Printf("Warning: No root thread found for UID %d", imapMsg.Uid)
			continue
		}

		// Get stable thread ID from the root message's Message-ID
		stableThreadID, ok := rootUIDToStableID[rootUID]
		if !ok {
			// Fallback: try to get from the root message if we have it
			if rootMsg, found := uidToMessageMap[rootUID]; found {
				if rootMsg.Envelope != nil && len(rootMsg.Envelope.MessageId) > 0 {
					stableThreadID = rootMsg.Envelope.MessageId
					rootUIDToStableID[rootUID] = stableThreadID
				}
			}
			if stableThreadID == "" {
				log.Printf("Warning: No Message-ID found for root UID %d", rootUID)
				continue
			}
		}

		// Get or create the thread
		threadModel, err := db.GetThreadByStableID(ctx, s.pool, userID, stableThreadID)
		if err != nil {
			if !errors.Is(err, db.ErrThreadNotFound) {
				return fmt.Errorf("failed to get thread: %w", err)
			}

			// Error IS ErrThreadNotFound, so we must create the thread.
			subject := ""
			if rootMsg, found := uidToMessageMap[rootUID]; found {
				if rootMsg.Envelope != nil {
					subject = rootMsg.Envelope.Subject
				}
			}
			threadModel = &models.Thread{
				UserID:         userID,
				StableThreadID: stableThreadID,
				Subject:        subject,
			}

			if err := db.SaveThread(ctx, s.pool, threadModel); err != nil {
				return fmt.Errorf("failed to save thread: %w", err)
			}
		}

		// Parse and save the message
		msg, err := ParseMessage(imapMsg, threadModel.ID, userID, folderName)
		if err != nil {
			log.Printf("Warning: Failed to parse message UID %d: %v", imapMsg.Uid, err)
			continue
		}

		if err := db.SaveMessage(ctx, s.pool, msg); err != nil {
			return fmt.Errorf("failed to save message: %w", err)
		}
	}

	// Set the folder sync timestamp after a successful sync
	if err := db.SetFolderSyncTimestamp(ctx, s.pool, userID, folderName); err != nil {
		log.Printf("Warning: Failed to set folder sync timestamp: %v", err)
		// Don't fail the entire sync if timestamp update fails
	}

	return nil
}

// SyncFullMessage syncs the full message body from IMAP.
func (s *Service) SyncFullMessage(ctx context.Context, userID, folderName string, imapUID int64) error {
	client, _, err := s.getClientAndSelectFolder(ctx, userID, folderName)
	if err != nil {
		return err
	}

	return s.syncSingleMessage(ctx, client, userID, folderName, imapUID)
}

// SyncFullMessages syncs multiple message bodies from IMAP in a batch.
// It groups messages by folder and syncs them efficiently to reduce network calls.
func (s *Service) SyncFullMessages(ctx context.Context, userID string, messages []MessageToSync) error {
	if len(messages) == 0 {
		return nil
	}

	// Group messages by folder to minimize folder SELECT operations
	folderToUIDs := make(map[string][]int64)
	for _, msg := range messages {
		folderToUIDs[msg.FolderName] = append(folderToUIDs[msg.FolderName], msg.IMAPUID)
	}

	// Get user settings and the password once
	settings, imapPassword, err := s.getSettingsAndPassword(ctx, userID)
	if err != nil {
		return err
	}

	// Sync messages grouped by folder
	for folderName, uids := range folderToUIDs {
		// Get IMAP client (internal use - need concrete type)
		client, err := s.clientPool.getClientConcrete(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword)
		if err != nil {
			log.Printf("Warning: Failed to get IMAP client for folder %s: %v", folderName, err)
			continue
		}

		// Select the folder once for all messages in this folder
		if _, err := client.Select(folderName, false); err != nil {
			log.Printf("Warning: Failed to select folder %s: %v", folderName, err)
			continue
		}

		// Sync each message in this folder
		for _, imapUID := range uids {
			if err := s.syncSingleMessage(ctx, client, userID, folderName, imapUID); err != nil {
				log.Printf("Warning: Failed to sync message UID %d in folder %s: %v", imapUID, folderName, err)
				// Continue with other messages
			}
		}
	}

	return nil
}

// syncSingleMessage syncs a single message body (helper for batch sync).
func (s *Service) syncSingleMessage(ctx context.Context, client *imapclient.Client, userID, folderName string, imapUID int64) error {
	// Fetch the full message
	imapMsg, err := FetchFullMessage(client, uint32(imapUID))
	if err != nil {
		return fmt.Errorf("failed to fetch full message: %w", err)
	}

	// Get existing message from DB
	msg, err := db.GetMessageByUID(ctx, s.pool, userID, folderName, imapUID)
	if err != nil {
		return fmt.Errorf("failed to get message from DB: %w", err)
	}

	// Parse body and update message
	parsedMsg, err := ParseMessage(imapMsg, msg.ThreadID, userID, folderName)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Update message with body
	msg.UnsafeBodyHTML = parsedMsg.UnsafeBodyHTML
	msg.BodyText = parsedMsg.BodyText

	// Save message with body
	if err := db.SaveMessage(ctx, s.pool, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Save attachments
	for _, att := range parsedMsg.Attachments {
		att.MessageID = msg.ID
		if err := db.SaveAttachment(ctx, s.pool, &att); err != nil {
			log.Printf("Warning: Failed to save attachment: %v", err)
		}
	}

	return nil
}

// ShouldSyncFolder checks if we should sync the folder based on cache TTL.
func (s *Service) ShouldSyncFolder(ctx context.Context, userID, folderName string) (bool, error) {
	syncTimestamp, err := db.GetFolderSyncTimestamp(ctx, s.pool, userID, folderName)
	if err != nil {
		return false, err
	}

	if syncTimestamp == nil {
		// No sync timestamp, need to sync
		return true, nil
	}

	age := time.Since(*syncTimestamp)
	return age > s.cacheTTL, nil
}

// Close closes the service and cleans up connections.
func (s *Service) Close() {
	s.clientPool.Close()
}
