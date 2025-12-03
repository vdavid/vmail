package api

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
)

// AttachmentHandler handles attachment download requests.
type AttachmentHandler struct {
	pool        *pgxpool.Pool
	imapService imap.IMAPService
}

// NewAttachmentHandler creates a new AttachmentHandler instance.
func NewAttachmentHandler(pool *pgxpool.Pool, imapService imap.IMAPService) *AttachmentHandler {
	return &AttachmentHandler{
		pool:        pool,
		imapService: imapService,
	}
}

// GetAttachment serves an attachment by fetching it from IMAP.
// Path: /api/v1/messages/{messageId}/attachments/{attachmentId}
// The attachmentId is URL-encoded and can be either a content_id or filename.
func (h *AttachmentHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx, w, h.pool)
	if !ok {
		return
	}

	// Parse path: /api/v1/messages/{messageId}/attachments/{attachmentId}
	messageID, attachmentID, err := parseAttachmentPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get message from database to get folder and IMAP UID
	msg, err := db.GetMessageByID(ctx, h.pool, messageID)
	if err != nil {
		log.Printf("AttachmentHandler: Failed to get message: %v", err)
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Verify the message belongs to this user
	if msg.UserID != userID {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Get the attachment metadata to determine how to find it
	attachment, err := db.GetAttachmentByID(ctx, h.pool, attachmentID)
	if err != nil {
		log.Printf("AttachmentHandler: Failed to get attachment: %v", err)
		http.Error(w, "Attachment not found", http.StatusNotFound)
		return
	}

	// Fetch attachment content from IMAP
	content, mimeType, filename, err := h.imapService.FetchAttachmentContent(
		ctx,
		userID,
		msg.IMAPFolderName,
		msg.IMAPUID,
		attachment.ContentID,
		attachment.Filename,
	)
	if err != nil {
		log.Printf("AttachmentHandler: Failed to fetch attachment: %v", err)
		http.Error(w, "Failed to fetch attachment", http.StatusInternalServerError)
		return
	}

	// Set response headers
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// For non-inline requests (download), set Content-Disposition
	if r.URL.Query().Get("download") == "true" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	} else {
		// For inline display
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.Header().Set("Cache-Control", "private, max-age=86400") // Cache for 24 hours

	if _, err := w.Write(content); err != nil {
		log.Printf("AttachmentHandler: Failed to write response: %v", err)
	}
}

// parseAttachmentPath extracts messageId and attachmentId from the path.
func parseAttachmentPath(path string) (messageID, attachmentID string, err error) {
	// Expected format: /api/v1/messages/{messageId}/attachments/{attachmentId}
	path = strings.TrimPrefix(path, "/api/v1/messages/")
	parts := strings.Split(path, "/attachments/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid path format")
	}

	messageID = parts[0]
	attachmentID, err = url.PathUnescape(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("invalid attachment ID encoding: %w", err)
	}

	return messageID, attachmentID, nil
}
