package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
)

// ThreadHandler handles individual thread-related API requests.
type ThreadHandler struct {
	pool        *pgxpool.Pool
	encryptor   *crypto.Encryptor
	imapService imap.IMAPService
}

// NewThreadHandler creates a new ThreadHandler instance.
func NewThreadHandler(pool *pgxpool.Pool, encryptor *crypto.Encryptor, imapService imap.IMAPService) *ThreadHandler {
	return &ThreadHandler{
		pool:        pool,
		encryptor:   encryptor,
		imapService: imapService,
	}
}

// getStableThreadIDFromPath extracts the stable thread ID from the request path.
// The thread ID is expected to be URL-encoded (percent-encoded) raw Message-ID.
func getStableThreadIDFromPath(path string) (string, error) {
	pathParts := strings.Split(strings.TrimPrefix(path, "/api/v1/thread/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		return "", fmt.Errorf("thread_id is required")
	}

	// URL decode the thread ID (handles %3C for <, %3E for >, etc.)
	decoded, err := url.PathUnescape(pathParts[0])
	if err != nil {
		return "", fmt.Errorf("invalid thread_id encoding: %w", err)
	}

	return decoded, nil
}

// collectMessagesToSync collects messages that need syncing and returns them with a UID-to-index map.
func collectMessagesToSync(messages []*models.Message) ([]imap.MessageToSync, map[int64]int) {
	messagesToSync := make([]imap.MessageToSync, 0)
	messageUIDToIndex := make(map[int64]int)
	for i, msg := range messages {
		if msg.UnsafeBodyHTML == "" && msg.BodyText == "" {
			messagesToSync = append(messagesToSync, imap.MessageToSync{
				FolderName: msg.IMAPFolderName,
				IMAPUID:    msg.IMAPUID,
			})
			messageUIDToIndex[msg.IMAPUID] = i
		}
	}
	return messagesToSync, messageUIDToIndex
}

// syncMissingBodies syncs missing message bodies and updates the messages slice.
func (h *ThreadHandler) syncMissingBodies(ctx context.Context, userID string, messages []*models.Message, messagesToSync []imap.MessageToSync, messageUIDToIndex map[int64]int) {
	if len(messagesToSync) == 0 {
		return
	}

	log.Printf("ThreadHandler: Syncing %d message bodies in batch", len(messagesToSync))
	if err := h.imapService.SyncFullMessages(ctx, userID, messagesToSync); err != nil {
		log.Printf("ThreadHandler: Failed to batch sync message bodies: %v", err)
		return
	}

	// Re-fetch all synced messages to get updated bodies
	for _, msgToSync := range messagesToSync {
		updatedMsg, err := db.GetMessageByUID(ctx, h.pool, userID, msgToSync.FolderName, msgToSync.IMAPUID)
		if err == nil {
			if idx, found := messageUIDToIndex[msgToSync.IMAPUID]; found {
				messages[idx] = updatedMsg
			}
		}
	}
}

// assignAttachments assigns attachments from the batch-fetched map to messages.
func assignAttachments(messages []*models.Message, attachmentsMap map[string][]*models.Attachment) {
	for _, msg := range messages {
		attachments := attachmentsMap[msg.ID]
		if attachments == nil {
			attachments = []*models.Attachment{}
		}

		// Convert []*Attachment to []Attachment
		msgAttachments := make([]models.Attachment, 0, len(attachments))
		for _, att := range attachments {
			if att != nil {
				msgAttachments = append(msgAttachments, *att)
			}
		}
		msg.Attachments = msgAttachments
	}
}

// convertMessagesToThreadMessages converts []*Message to []Message.
// Ensures that Attachments is always an array, never nil.
func convertMessagesToThreadMessages(messages []*models.Message) []models.Message {
	threadMessages := make([]models.Message, 0, len(messages))
	for _, msg := range messages {
		if msg != nil {
			// Ensure Attachments is always initialized to an empty array if nil
			if msg.Attachments == nil {
				msg.Attachments = []models.Attachment{}
			}
			threadMessages = append(threadMessages, *msg)
		}
	}
	return threadMessages
}

// GetThread returns a single email thread with all its messages.
func (h *ThreadHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx, w, h.pool)
	if !ok {
		return
	}

	// Get thread_id from the URL path
	stableThreadID, err := getStableThreadIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get thread from the database
	thread, err := db.GetThreadByStableID(ctx, h.pool, userID, stableThreadID)
	if err != nil {
		if errors.Is(err, db.ErrThreadNotFound) {
			http.Error(w, "Thread not found", http.StatusNotFound)
			return
		}
		log.Printf("ThreadHandler: Failed to get thread: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get messages for thread
	messages, err := db.GetMessagesForThread(ctx, h.pool, thread.ID)
	if err != nil {
		log.Printf("ThreadHandler: Failed to get messages: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Ensure messages is not nil (defensive check for static analysis)
	if messages == nil {
		messages = []*models.Message{}
	}

	// Collect all message IDs for batch attachment fetching
	messageIDs := make([]string, 0, len(messages))
	for _, msg := range messages {
		messageIDs = append(messageIDs, msg.ID)
	}

	// Fetch all attachments in a single query (fixes N+1 query bug)
	attachmentsMap, err := db.GetAttachmentsForMessages(ctx, h.pool, messageIDs)
	if err != nil {
		log.Printf("ThreadHandler: Failed to get attachments: %v", err)
		attachmentsMap = make(map[string][]*models.Attachment)
	}

	// Collect messages that need syncing and sync them
	messagesToSync, messageUIDToIndex := collectMessagesToSync(messages)
	h.syncMissingBodies(ctx, userID, messages, messagesToSync, messageUIDToIndex)

	// Assign attachments and convert messages
	assignAttachments(messages, attachmentsMap)
	thread.Messages = convertMessagesToThreadMessages(messages)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(thread); err != nil {
		log.Printf("ThreadHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
