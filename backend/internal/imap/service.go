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
// The IMAP pool is injected so that a single shared pool can be used across
// handlers and services, ensuring per-user connection limits are enforced
// consistently.
type Service struct {
	dbPool    *pgxpool.Pool
	imapPool  IMAPPool
	encryptor *crypto.Encryptor
	cacheTTL  time.Duration
}

// NewService creates a new IMAP service.
func NewService(dbPool *pgxpool.Pool, imapPool IMAPPool, encryptor *crypto.Encryptor) *Service {
	return &Service{
		dbPool:    dbPool,
		imapPool:  imapPool,
		encryptor: encryptor,
		cacheTTL:  5 * time.Minute, // Default cache TTL
	}
}

// getSettingsAndPassword gets user settings and decrypts the IMAP password.
func (s *Service) getSettingsAndPassword(ctx context.Context, userID string) (*models.UserSettings, string, error) {
	settings, err := db.GetUserSettings(ctx, s.dbPool, userID)
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
// Thread-safe: The connection is locked during folder selection to prevent concurrent folder selections
// from interfering with each other. The connection will be automatically unlocked after the operation.
func (s *Service) getClientAndSelectFolder(ctx context.Context, userID, folderName string) (*imapclient.Client, *imap.MailboxStatus, error) {
	settings, imapPassword, err := s.getSettingsAndPassword(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	// Get IMAP client (internal use - need concrete type)
	// The connection is locked when returned, ensuring thread-safe folder selection.
	clientIface, release, err := s.imapPool.GetClient(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get IMAP client: %w", err)
	}
	defer release()

	wrapper, ok := clientIface.(*ClientWrapper)
	if !ok || wrapper.client == nil {
		return nil, nil, fmt.Errorf("failed to unwrap IMAP client")
	}
	client := wrapper.client

	// Select the folder - connection is locked, so this is thread-safe
	// Even if multiple goroutines call this concurrently, they will use different connections
	// from the pool, or the same connection will be serialized by the lock
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

// buildUIDToMessageMap builds a map of UID to message for a quick lookup.
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
	threadModel, err := db.GetThreadByStableID(ctx, s.dbPool, userID, stableThreadID)
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

		if err := db.SaveThread(ctx, s.dbPool, threadModel); err != nil {
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

	if err := db.SaveMessage(ctx, s.dbPool, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// incrementalSyncResult holds the result of attempting an incremental sync.
type incrementalSyncResult struct {
	uidsToSync   []uint32
	highestUID   uint32
	shouldReturn bool // true if we should return early (no new messages)
}

// tryIncrementalSync attempts to perform an incremental sync.
// Returns the result and whether incremental sync was successful.
func (s *Service) tryIncrementalSync(ctx context.Context, client *imapclient.Client, userID, folderName string, syncInfo *db.FolderSyncInfo) (incrementalSyncResult, bool) {
	if syncInfo == nil || syncInfo.LastSyncedUID == nil || *syncInfo.LastSyncedUID <= 0 {
		return incrementalSyncResult{}, false
	}

	lastUID := uint32(*syncInfo.LastSyncedUID)
	log.Printf("Incremental sync: fetching UIDs >= %d", lastUID+1)

	newUIDs, err := SearchUIDsSince(client, lastUID+1)
	if err != nil {
		log.Printf("Warning: Failed to search for new UIDs, falling back to full sync: %v", err)
		return incrementalSyncResult{}, false
	}

	if len(newUIDs) == 0 {
		log.Printf("No new messages to sync")
		// Update sync timestamp even though there's nothing new
		if err := db.SetFolderSyncInfo(ctx, s.dbPool, userID, folderName, syncInfo.LastSyncedUID); err != nil {
			log.Printf("Warning: Failed to update folder sync timestamp: %v", err)
		}
		// Trigger background thread count update
		go s.updateThreadCountInBackground(userID, folderName)
		return incrementalSyncResult{shouldReturn: true}, true
	}

	log.Printf("Found %d new messages to sync", len(newUIDs))

	// Find the highest UID
	var highestUID uint32
	for _, uid := range newUIDs {
		if uid > highestUID {
			highestUID = uid
		}
	}

	return incrementalSyncResult{
		uidsToSync:   newUIDs,
		highestUID:   highestUID,
		shouldReturn: false,
	}, true
}

// fullSyncResult holds the result of performing a full sync.
type fullSyncResult struct {
	threadMaps   *threadMaps
	uidsToSync   []uint32
	highestUID   uint32
	shouldReturn bool // true if we should return early (no messages)
}

// performFullSync performs a full sync of all threads in the folder.
// Falls back to fetching all UIDs using SEARCH if THREAD command is not supported.
func (s *Service) performFullSync(ctx context.Context, client *imapclient.Client, userID, folderName string) (fullSyncResult, error) {
	log.Printf("Full sync: fetching all threads")
	threads, err := RunThreadCommand(client)
	if err != nil {
		// THREAD command not supported (e.g., by test IMAP server) - fall back to SEARCH
		log.Printf("THREAD command not supported, falling back to SEARCH: %v", err)
		// Fetch all UIDs using SEARCH (starting from UID 1)
		uidsToSync, err := SearchUIDsSince(client, 1)
		if err != nil {
			return fullSyncResult{}, fmt.Errorf("failed to search for all UIDs: %w", err)
		}

		if len(uidsToSync) == 0 {
			log.Printf("No messages found in folder %s", folderName)
			// Still update sync info
			if err := db.SetFolderSyncInfo(ctx, s.dbPool, userID, folderName, nil); err != nil {
				log.Printf("Warning: Failed to set folder sync info: %v", err)
			}
			return fullSyncResult{shouldReturn: true}, nil
		}

		// Find the highest UID
		var highestUID uint32
		for _, uid := range uidsToSync {
			if uid > highestUID {
				highestUID = uid
			}
		}

		// Return without threadMaps (will be nil) - messages will be processed without threading
		return fullSyncResult{
			threadMaps:   nil, // No thread structure available
			uidsToSync:   uidsToSync,
			highestUID:   highestUID,
			shouldReturn: false,
		}, nil
	}

	log.Printf("Found %d threads in folder %s", len(threads), folderName)

	threadMaps := buildThreadMaps(threads)
	uidsToSync := threadMaps.allUIDs

	if len(uidsToSync) == 0 {
		log.Printf("No messages found in folder %s", folderName)
		// Still update sync info
		if err := db.SetFolderSyncInfo(ctx, s.dbPool, userID, folderName, nil); err != nil {
			log.Printf("Warning: Failed to set folder sync info: %v", err)
		}
		return fullSyncResult{shouldReturn: true}, nil
	}

	// Find the highest UID
	var highestUID uint32
	for _, uid := range uidsToSync {
		if uid > highestUID {
			highestUID = uid
		}
	}

	return fullSyncResult{
		threadMaps:   threadMaps,
		uidsToSync:   uidsToSync,
		highestUID:   highestUID,
		shouldReturn: false,
	}, nil
}

// processIncrementalMessages processes messages during incremental sync.
func (s *Service) processIncrementalMessages(ctx context.Context, messages []*imap.Message, userID, folderName string) {
	for _, imapMsg := range messages {
		if err := s.processIncrementalMessage(ctx, imapMsg, userID, folderName); err != nil {
			log.Printf("Warning: Failed to process message UID %d: %v", imapMsg.Uid, err)
			// Continue with other messages
		}
	}
}

// processFullSyncMessages processes messages during full sync using thread structure.
func (s *Service) processFullSyncMessages(ctx context.Context, messages []*imap.Message, threadMaps *threadMaps, userID, folderName string) error {
	threadMaps.uidToMessage = buildUIDToMessageMap(messages)
	threadMaps.rootUIDToStableID = buildRootUIDToStableIDMap(threadMaps.rootUIDs, threadMaps.uidToMessage)

	for _, imapMsg := range messages {
		rootUID, ok := threadMaps.uidToThreadRoot[imapMsg.Uid]
		if !ok {
			log.Printf("Warning: No root thread found for UID %d", imapMsg.Uid)
			continue
		}

		stableThreadID := getStableThreadID(rootUID, threadMaps.rootUIDToStableID, threadMaps.uidToMessage)
		if stableThreadID == "" {
			log.Printf("Warning: No Message-ID found for root UID %d", rootUID)
			continue
		}

		if err := s.processMessage(ctx, imapMsg, rootUID, stableThreadID, userID, folderName, threadMaps.uidToMessage); err != nil {
			return err
		}
	}

	return nil
}

// SyncThreadsForFolder syncs threads from IMAP for a specific folder.
// Uses incremental sync if possible (only syncs new messages since last sync).
func (s *Service) SyncThreadsForFolder(ctx context.Context, userID, folderName string) error {
	client, mbox, err := s.getClientAndSelectFolder(ctx, userID, folderName)
	if err != nil {
		return err
	}

	log.Printf("Selected folder %s: %d messages", folderName, mbox.Messages)

	// Check if we can do incremental sync
	syncInfo, err := db.GetFolderSyncInfo(ctx, s.dbPool, userID, folderName)
	if err != nil {
		log.Printf("Warning: Failed to get folder sync info: %v", err)
		syncInfo = nil // Fall back to full sync
	}

	// Try incremental sync first
	incResult, isIncremental := s.tryIncrementalSync(ctx, client, userID, folderName, syncInfo)
	if isIncremental {
		if incResult.shouldReturn {
			return nil
		}
		// Incremental sync path: process messages without thread structure
		messages, err := FetchMessageHeaders(client, incResult.uidsToSync)
		if err != nil {
			return fmt.Errorf("failed to fetch message headers: %w", err)
		}
		log.Printf("Fetched %d message headers", len(messages))
		s.processIncrementalMessages(ctx, messages, userID, folderName)

		// Update sync info with the highest UID
		highestUIDInt64 := int64(incResult.highestUID)
		if err := db.SetFolderSyncInfo(ctx, s.dbPool, userID, folderName, &highestUIDInt64); err != nil {
			log.Printf("Warning: Failed to set folder sync info: %v", err)
		}
		go s.updateThreadCountInBackground(userID, folderName)
		return nil
	}

	// Full sync path: get thread structure first
	fullResult, err := s.performFullSync(ctx, client, userID, folderName)
	if err != nil {
		return err
	}
	if fullResult.shouldReturn {
		return nil
	}

	// Fetch message headers for UIDs we need to sync
	messages, err := FetchMessageHeaders(client, fullResult.uidsToSync)
	if err != nil {
		return fmt.Errorf("failed to fetch message headers: %w", err)
	}

	log.Printf("Fetched %d message headers", len(messages))

	// Process messages: use thread structure if available, otherwise use incremental processing
	threadMaps := fullResult.threadMaps
	if threadMaps == nil {
		// THREAD command not supported - process messages without thread structure
		// (same as incremental sync)
		s.processIncrementalMessages(ctx, messages, userID, folderName)
	} else {
		// Process messages using thread structure
		if err := s.processFullSyncMessages(ctx, messages, threadMaps, userID, folderName); err != nil {
			return err
		}
	}

	// Update sync info with the highest UID
	highestUIDInt64 := int64(fullResult.highestUID)
	if err := db.SetFolderSyncInfo(ctx, s.dbPool, userID, folderName, &highestUIDInt64); err != nil {
		log.Printf("Warning: Failed to set folder sync info: %v", err)
		// Don't fail the entire sync if timestamp update fails
	}

	// Trigger background thread count update
	go s.updateThreadCountInBackground(userID, folderName)

	return nil
}

// processIncrementalMessage processes a single message during incremental sync.
// It matches the message to an existing thread or creates a new one.
// For simplicity, we use the message's own Message-ID to match threads.
// If the Message-ID matches a thread's stable ID, it's the root message of that thread.
// Otherwise, we create a new thread. Full sync will correct any threading issues.
func (s *Service) processIncrementalMessage(ctx context.Context, imapMsg *imap.Message, userID, folderName string) error {
	if imapMsg.Envelope == nil || len(imapMsg.Envelope.MessageId) == 0 {
		log.Printf("Warning: Message UID %d has no Message-ID, skipping", imapMsg.Uid)
		return nil
	}

	messageID := imapMsg.Envelope.MessageId

	// For incremental sync, we use a simplified approach:
	// 1. Try to find a thread where this Message-ID is the stable thread ID (root message)
	// 2. If not found, check if this message is already in the DB (might be a reply)
	// 3. If still not found, create a new thread with this Message-ID as root
	// Note: This is a simplification - full sync will correct threading using THREAD command

	// First, try to find the thread by Message-ID (this works for root messages)
	threadModel, err := db.GetThreadByStableID(ctx, s.dbPool, userID, messageID)
	if err != nil {
		if !errors.Is(err, db.ErrThreadNotFound) {
			return fmt.Errorf("failed to get thread: %w", err)
		}

		// Thread was not found - check if this message already exists (might be a reply to an existing thread)
		existingMsg, err := db.GetMessageByMessageID(ctx, s.dbPool, userID, messageID)
		if err == nil && existingMsg != nil {
			// Message already exists, get its thread
			threadModel, err = db.GetThreadByID(ctx, s.dbPool, existingMsg.ThreadID)
			if err != nil {
				return fmt.Errorf("failed to get existing message's thread: %w", err)
			}
		} else {
			// New message - create a new thread
			// For incremental sync, we'll use this message's Message-ID as the stable ID
			// Full sync will correct this if it's actually a reply
			threadModel = &models.Thread{
				UserID:         userID,
				StableThreadID: messageID,
				Subject:        "",
			}
			if imapMsg.Envelope != nil {
				threadModel.Subject = imapMsg.Envelope.Subject
			}
			if err := db.SaveThread(ctx, s.dbPool, threadModel); err != nil {
				return fmt.Errorf("failed to save thread: %w", err)
			}
		}
	}

	// Parse and save the message
	msg, err := ParseMessage(imapMsg, threadModel.ID, userID, folderName)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	if err := db.SaveMessage(ctx, s.dbPool, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// updateThreadCountInBackground updates the thread count in the background.
// Uses a 30-second timeout to avoid hanging indefinitely.
func (s *Service) updateThreadCountInBackground(userID, folderName string) {
	// Use a new context with timeout to avoid hanging
	bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.UpdateThreadCount(bgCtx, s.dbPool, userID, folderName); err != nil {
		log.Printf("Warning: Failed to update thread count in background for folder %s: %v", folderName, err)
	} else {
		log.Printf("Updated thread count for folder %s", folderName)
	}
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
// Thread-safe: Each folder selection uses a locked connection from the pool, ensuring
// that concurrent syncs for the same user use different connections or are serialized.
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
		clientIface, release, err := s.imapPool.GetClient(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword)
		if err != nil {
			log.Printf("Warning: Failed to get IMAP client for folder %s: %v", folderName, err)
			continue
		}

		wrapper, ok := clientIface.(*ClientWrapper)
		if !ok || wrapper.client == nil {
			log.Printf("Warning: Failed to unwrap IMAP client for folder %s", folderName)
			release()
			continue
		}

		client := wrapper.client

		// Select the folder once for all messages in this folder
		if _, err := client.Select(folderName, false); err != nil {
			log.Printf("Warning: Failed to select folder %s: %v", folderName, err)
			release()
			continue
		}

		// Sync each message in this folder
		for _, imapUID := range uids {
			if err := s.syncSingleMessage(ctx, client, userID, folderName, imapUID); err != nil {
				log.Printf("Warning: Failed to sync message UID %d in folder %s: %v", imapUID, folderName, err)
				// Continue with other messages
			}
		}

		release()
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
	msg, err := db.GetMessageByUID(ctx, s.dbPool, userID, folderName, imapUID)
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
	if err := db.SaveMessage(ctx, s.dbPool, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Save attachments
	for _, att := range parsedMsg.Attachments {
		att.MessageID = msg.ID
		if err := db.SaveAttachment(ctx, s.dbPool, &att); err != nil {
			log.Printf("Warning: Failed to save attachment: %v", err)
		}
	}

	return nil
}

// ShouldSyncFolder checks if we should sync the folder based on cache TTL.
func (s *Service) ShouldSyncFolder(ctx context.Context, userID, folderName string) (bool, error) {
	syncInfo, err := db.GetFolderSyncInfo(ctx, s.dbPool, userID, folderName)
	if err != nil {
		return false, err
	}

	if syncInfo == nil || syncInfo.SyncedAt == nil {
		// No sync timestamp, need to sync
		return true, nil
	}

	age := time.Since(*syncInfo.SyncedAt)
	return age > s.cacheTTL, nil
}

// Close closes the service and cleans up connections.
func (s *Service) Close() {
	s.imapPool.Close()
}
