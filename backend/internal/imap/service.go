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

// threadMaps contains the maps needed for thread processing.
type threadMaps struct {
	allUIDs           []uint32
	uidToThreadRoot   map[uint32]uint32
	rootUIDs          []uint32
	uidToMessage      map[uint32]*imap.Message
	rootUIDToStableID map[uint32]string
}

// buildThreadMaps builds all the maps needed for thread processing.
func buildThreadMaps(threads []*sortthread.Thread) *threadMaps {
	maps := &threadMaps{
		allUIDs:           make([]uint32, 0),
		uidToThreadRoot:   make(map[uint32]uint32),
		rootUIDs:          make([]uint32, 0),
		uidToMessage:      make(map[uint32]*imap.Message),
		rootUIDToStableID: make(map[uint32]string),
	}

	// Recursively map all messages in a thread to their root UID
	var mapThreadToRoot func(*sortthread.Thread, uint32)
	mapThreadToRoot = func(thread *sortthread.Thread, rootUID uint32) {
		if thread == nil {
			return
		}
		maps.uidToThreadRoot[thread.Id] = rootUID
		maps.allUIDs = append(maps.allUIDs, thread.Id)
		for _, child := range thread.Children {
			mapThreadToRoot(child, rootUID)
		}
	}

	// Build maps: find root for each top-level thread
	for _, thread := range threads {
		if thread == nil {
			continue
		}
		rootUID := thread.Id
		maps.rootUIDs = append(maps.rootUIDs, rootUID)
		mapThreadToRoot(thread, rootUID)
	}

	return maps
}

// buildUIDToMessageMap builds a map of UID to message for quick lookup.
func buildUIDToMessageMap(messages []*imap.Message) map[uint32]*imap.Message {
	uidToMessageMap := make(map[uint32]*imap.Message)
	for _, msg := range messages {
		uidToMessageMap[msg.Uid] = msg
	}
	return uidToMessageMap
}

// buildRootUIDToStableIDMap builds a map of root UID to stable thread ID.
func buildRootUIDToStableIDMap(rootUIDs []uint32, uidToMessageMap map[uint32]*imap.Message) map[uint32]string {
	rootUIDToStableID := make(map[uint32]string)
	for _, rootUID := range rootUIDs {
		if rootMsg, found := uidToMessageMap[rootUID]; found {
			if rootMsg.Envelope != nil && len(rootMsg.Envelope.MessageId) > 0 {
				rootUIDToStableID[rootUID] = rootMsg.Envelope.MessageId
			}
		}
	}
	return rootUIDToStableID
}

// getStableThreadID gets the stable thread ID for a root UID, with fallback.
func getStableThreadID(rootUID uint32, rootUIDToStableID map[uint32]string, uidToMessageMap map[uint32]*imap.Message) string {
	stableThreadID, ok := rootUIDToStableID[rootUID]
	if !ok {
		// Fallback: try to get from the root message if we have it
		if rootMsg, found := uidToMessageMap[rootUID]; found {
			if rootMsg.Envelope != nil && len(rootMsg.Envelope.MessageId) > 0 {
				stableThreadID = rootMsg.Envelope.MessageId
				rootUIDToStableID[rootUID] = stableThreadID
			}
		}
	}
	return stableThreadID
}

// getOrCreateThread gets an existing thread or creates a new one.
func (s *Service) getOrCreateThread(ctx context.Context, userID, stableThreadID string, rootUID uint32, uidToMessageMap map[uint32]*imap.Message) (*models.Thread, error) {
	threadModel, err := db.GetThreadByStableID(ctx, s.pool, userID, stableThreadID)
	if err != nil {
		if !errors.Is(err, db.ErrThreadNotFound) {
			return nil, fmt.Errorf("failed to get thread: %w", err)
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
			return nil, fmt.Errorf("failed to save thread: %w", err)
		}
	}
	return threadModel, nil
}

// processMessage processes a single message and saves it to the database.
func (s *Service) processMessage(ctx context.Context, imapMsg *imap.Message, rootUID uint32, stableThreadID string, userID, folderName string, uidToMessageMap map[uint32]*imap.Message) error {
	threadModel, err := s.getOrCreateThread(ctx, userID, stableThreadID, rootUID, uidToMessageMap)
	if err != nil {
		return err
	}

	// Parse and save the message
	msg, err := ParseMessage(imapMsg, threadModel.ID, userID, folderName)
	if err != nil {
		log.Printf("Warning: Failed to parse message UID %d: %v", imapMsg.Uid, err)
		return nil // Continue processing other messages
	}

	if err := db.SaveMessage(ctx, s.pool, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
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

	// Build thread maps
	threadMaps := buildThreadMaps(threads)

	if len(threadMaps.allUIDs) == 0 {
		log.Printf("No messages found in folder %s", folderName)
		return nil
	}

	// Fetch message headers for all messages
	messages, err := FetchMessageHeaders(client, threadMaps.allUIDs)
	if err != nil {
		return fmt.Errorf("failed to fetch message headers: %w", err)
	}

	// Build maps
	threadMaps.uidToMessage = buildUIDToMessageMap(messages)
	threadMaps.rootUIDToStableID = buildRootUIDToStableIDMap(threadMaps.rootUIDs, threadMaps.uidToMessage)

	log.Printf("Fetched %d message headers", len(messages))

	// Process each message
	for _, imapMsg := range messages {
		rootUID, ok := threadMaps.uidToThreadRoot[imapMsg.Uid]
		if !ok {
			log.Printf("Warning: No root thread found for UID %d", imapMsg.Uid)
			continue
		}

		// Get stable thread ID from the root message's Message-ID
		stableThreadID := getStableThreadID(rootUID, threadMaps.rootUIDToStableID, threadMaps.uidToMessage)
		if stableThreadID == "" {
			log.Printf("Warning: No Message-ID found for root UID %d", rootUID)
			continue
		}

		if err := s.processMessage(ctx, imapMsg, rootUID, stableThreadID, userID, folderName, threadMaps.uidToMessage); err != nil {
			return err
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
