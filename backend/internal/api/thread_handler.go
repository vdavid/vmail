package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
)

type ThreadHandler struct {
	pool        *pgxpool.Pool
	encryptor   *crypto.Encryptor
	imapService imap.IMAPService
}

func NewThreadHandler(pool *pgxpool.Pool, encryptor *crypto.Encryptor, imapService imap.IMAPService) *ThreadHandler {
	return &ThreadHandler{
		pool:        pool,
		encryptor:   encryptor,
		imapService: imapService,
	}
}

func (h *ThreadHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := h.getUserIDFromContext(ctx, w)
	if !ok {
		return
	}

	// Get thread_id from the URL path.
	// Path should be /api/v1/thread/{thread_id}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/thread/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "thread_id is required", http.StatusBadRequest)
		return
	}

	stableThreadID := pathParts[0]

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

	// Collect all message IDs for batch attachment fetching
	messageIDs := make([]string, 0, len(messages))
	for _, msg := range messages {
		messageIDs = append(messageIDs, msg.ID)
	}

	// Fetch all attachments in a single query (fixes N+1 query bug)
	attachmentsMap, err := db.GetAttachmentsForMessages(ctx, h.pool, messageIDs)
	if err != nil {
		log.Printf("ThreadHandler: Failed to get attachments: %v", err)
		// Continue anyway - attachments will be empty
		attachmentsMap = make(map[string][]*models.Attachment)
	}

	// Collect messages that need syncing (fixes N+1 network bug)
	messagesToSync := make([]imap.MessageToSync, 0)
	messageUIDToIndex := make(map[int64]int) // Map UID to index in messages slice
	for i, msg := range messages {
		if msg.UnsafeBodyHTML == "" && msg.BodyText == "" {
			messagesToSync = append(messagesToSync, imap.MessageToSync{
				FolderName: msg.IMAPFolderName,
				IMAPUID:    msg.IMAPUID,
			})
			messageUIDToIndex[msg.IMAPUID] = i
		}
	}

	// Batch sync all missing messages
	if len(messagesToSync) > 0 {
		log.Printf("ThreadHandler: Syncing %d message bodies in batch", len(messagesToSync))
		if err := h.imapService.SyncFullMessages(ctx, userID, messagesToSync); err != nil {
			log.Printf("ThreadHandler: Failed to batch sync message bodies: %v", err)
			// Continue anyway - we'll return what we have
		} else {
			// Re-fetch all synced messages to get updated bodies
			for _, msgToSync := range messagesToSync {
				updatedMsg, err := db.GetMessageByUID(ctx, h.pool, userID, msgToSync.FolderName, msgToSync.IMAPUID)
				if err == nil {
					// Replace the message in the list
					if idx, found := messageUIDToIndex[msgToSync.IMAPUID]; found {
						messages[idx] = updatedMsg
					}
				}
			}
		}
	}

	// Assign attachments from the batch-fetched map
	for _, msg := range messages {
		// Get attachments from the batch-fetched map
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

	// Convert []*Message to []Message
	threadMessages := make([]models.Message, 0, len(messages))
	for _, msg := range messages {
		if msg != nil {
			threadMessages = append(threadMessages, *msg)
		}
	}
	thread.Messages = threadMessages

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(thread); err != nil {
		log.Printf("ThreadHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *ThreadHandler) getUserIDFromContext(ctx context.Context, w http.ResponseWriter) (string, bool) {
	email, ok := auth.GetUserEmailFromContext(ctx)
	if !ok {
		log.Println("ThreadHandler: No user email in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}

	userID, err := db.GetOrCreateUser(ctx, h.pool, email)
	if err != nil {
		log.Printf("ThreadHandler: Failed to get/create user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return "", false
	}

	return userID, true
}
