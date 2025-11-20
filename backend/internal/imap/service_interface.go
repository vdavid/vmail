package imap

import (
	"context"

	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/websocket"
)

// MessageToSync represents a message that needs to be synced.
type MessageToSync struct {
	FolderName string
	IMAPUID    int64
}

// IMAPService defines the interface for IMAP operations.
// This interface allows handlers to be tested with mock implementations.
// Note: The stutter in the naming is intentional because we have a struct called Service.
//
//goland:noinspection GoNameStartsWithPackageName
type IMAPService interface {
	// ShouldSyncFolder checks if we should sync the folder based on cache TTL.
	ShouldSyncFolder(ctx context.Context, userID, folderName string) (bool, error)

	// SyncThreadsForFolder syncs threads from IMAP for a specific folder.
	SyncThreadsForFolder(ctx context.Context, userID, folderName string) error

	// SyncFullMessage syncs the full message body from IMAP.
	SyncFullMessage(ctx context.Context, userID, folderName string, imapUID int64) error

	// SyncFullMessages syncs multiple message bodies from IMAP in a batch.
	// Messages are grouped by folder and synced efficiently.
	SyncFullMessages(ctx context.Context, userID string, messages []MessageToSync) error

	// Search searches for threads matching the query.
	// Returns threads, total count, and error.
	Search(ctx context.Context, userID string, query string, page, limit int) ([]*models.Thread, int, error)

	// StartIdleListener runs an IMAP IDLE loop for a user and pushes events to the WebSocket hub.
	// This function blocks until the context is cancelled.
	StartIdleListener(ctx context.Context, userID string, hub *websocket.Hub)

	// Close closes the service and cleans up connections.
	Close()
}

// Ensure Service implements IMAPService interface
var _ IMAPService = (*Service)(nil)
