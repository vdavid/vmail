package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
)

// FoldersHandler handles IMAP folder-related API requests.
type FoldersHandler struct {
	pool      *pgxpool.Pool
	encryptor *crypto.Encryptor
	imapPool  imap.IMAPPool
}

// NewFoldersHandler creates a new FoldersHandler instance.
func NewFoldersHandler(pool *pgxpool.Pool, encryptor *crypto.Encryptor, imapPool imap.IMAPPool) *FoldersHandler {
	return &FoldersHandler{
		pool:      pool,
		encryptor: encryptor,
		imapPool:  imapPool,
	}
}

// GetFolders returns the list of IMAP folders for the current user.
func (h *FoldersHandler) GetFolders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx, w, h.pool)
	if !ok {
		return
	}

	settings, imapPassword, ok := h.getUserSettingsAndPassword(ctx, w, userID)
	if !ok {
		return
	}

	// Use WithClient to ensure the client is always released
	err := h.imapPool.WithClient(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword, func(client imap.IMAPClient) error {
		folders, err := client.ListFolders()
		if err != nil {
			return h.handleListFoldersError(w, userID, err, settings, imapPassword)
		}

		h.writeFoldersResponse(w, folders)
		return nil
	})

	if err != nil {
		// Error handling is done inside the callback, so if we get here it's a connection error
		h.handleConnectionError(w, err)
	}
}

// getUserSettingsAndPassword retrieves user settings and decrypts the IMAP password.
func (h *FoldersHandler) getUserSettingsAndPassword(ctx context.Context, w http.ResponseWriter, userID string) (*models.UserSettings, string, bool) {
	settings, err := db.GetUserSettings(ctx, h.pool, userID)
	if err != nil {
		if errors.Is(err, db.ErrUserSettingsNotFound) {
			http.Error(w, "User settings not found", http.StatusNotFound)
			return nil, "", false
		}
		log.Printf("FoldersHandler: Failed to get user settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil, "", false
	}

	imapPassword, err := h.encryptor.Decrypt(settings.EncryptedIMAPPassword)
	if err != nil {
		log.Printf("FoldersHandler: Failed to decrypt IMAP password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return nil, "", false
	}

	return settings, imapPassword, true
}

// handleConnectionError handles errors when getting a client from the pool.
func (h *FoldersHandler) handleConnectionError(w http.ResponseWriter, err error) {
	log.Printf("FoldersHandler: Failed to get IMAP client: %v", err)
	errMsg := err.Error()
	if strings.Contains(errMsg, "i/o timeout") {
		http.Error(w, "Connection to IMAP server timed out. Please double-check your server hostname in your Settings and try again.", http.StatusServiceUnavailable)
	} else {
		http.Error(w, "Failed to connect to IMAP server", http.StatusInternalServerError)
	}
}

// handleListFoldersError handles errors from ListFolders, including retry logic.
// Returns an error to propagate to the WithClient callback.
func (h *FoldersHandler) handleListFoldersError(w http.ResponseWriter, userID string, err error, settings *models.UserSettings, imapPassword string) error {
	log.Printf("FoldersHandler: Failed to list folders: %v", err)
	errMsg := err.Error()

	if strings.Contains(errMsg, "SPECIAL-USE") {
		http.Error(w, "Your IMAP server doesn't support the SPECIAL-USE extension (RFC 6154), which is required for V-Mail to identify folder types. Please contact your email provider or use a different IMAP server.", http.StatusBadRequest)
		return err // Return error to stop processing
	}

	if h.isBrokenConnectionError(errMsg) {
		return h.retryListFolders(w, userID, settings, imapPassword)
	}

	http.Error(w, "Failed to list folders", http.StatusInternalServerError)
	return err // Return error to stop processing
}

// isBrokenConnectionError checks if the error message indicates a broken connection
// that can be recovered by retrying with a fresh IMAP client.
func (h *FoldersHandler) isBrokenConnectionError(errMsg string) bool {
	return strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "EOF")
}

// retryListFolders retries listing folders after removing the broken connection from the pool.
// This handles transient connection issues by getting a fresh IMAP client and retrying the operation.
// Returns an error to propagate to the WithClient callback.
func (h *FoldersHandler) retryListFolders(w http.ResponseWriter, userID string, settings *models.UserSettings, imapPassword string) error {
	h.imapPool.RemoveClient(userID)

	// Use WithClient for the retry to ensure release happens
	return h.imapPool.WithClient(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword, func(client imap.IMAPClient) error {
		folders, err := client.ListFolders()
		if err != nil {
			log.Printf("FoldersHandler: Failed to list folders on retry: %v", err)
			if strings.Contains(err.Error(), "SPECIAL-USE") {
				http.Error(w, "Your IMAP server doesn't support the SPECIAL-USE extension (RFC 6154), which is required for V-Mail to identify folder types. Please contact your email provider or use a different IMAP server.", http.StatusBadRequest)
				return err
			}
			http.Error(w, "Failed to list folders", http.StatusInternalServerError)
			return err
		}

		h.writeFoldersResponse(w, folders)
		return nil
	})
}

// writeFoldersResponse writes the folders response as JSON.
// Uses a buffered approach to prevent partial writes if JSON encoding fails.
func (h *FoldersHandler) writeFoldersResponse(w http.ResponseWriter, folders []*models.Folder) {
	sortFoldersByRole(folders)

	folderValues := make([]models.Folder, len(folders))
	for i, f := range folders {
		folderValues[i] = *f
	}

	if !WriteJSONResponse(w, folderValues) {
		return
	}
}

// sortFoldersByRole sorts folders by role priority, then alphabetically for "other" folders.
// Priority order: inbox, sent, drafts, spam, trash, archive, other (alphabetically).
func sortFoldersByRole(folders []*models.Folder) {
	rolePriority := map[string]int{
		"inbox":   1,
		"sent":    2,
		"drafts":  3,
		"spam":    4,
		"trash":   5,
		"archive": 6,
		"other":   7,
	}

	sort.Slice(folders, func(i, j int) bool {
		priorityI := rolePriority[folders[i].Role]
		priorityJ := rolePriority[folders[j].Role]

		if priorityI != priorityJ {
			return priorityI < priorityJ
		}

		// Same priority - sort alphabetically by name
		return folders[i].Name < folders[j].Name
	})
}
