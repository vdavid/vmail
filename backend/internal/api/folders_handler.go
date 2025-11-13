package api

import (
	"bytes"
	"context"
	"encoding/json"
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

	client, ok := h.getIMAPClient(w, userID, settings, imapPassword)
	if !ok {
		return
	}

	folders, ok := h.listFoldersWithRetry(w, userID, client, settings, imapPassword)
	if !ok {
		return
	}

	h.writeFoldersResponse(w, folders)
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

// getIMAPClient gets an IMAP client, handling connection errors.
func (h *FoldersHandler) getIMAPClient(w http.ResponseWriter, userID string, settings *models.UserSettings, imapPassword string) (imap.IMAPClient, bool) {
	client, err := h.imapPool.GetClient(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword)
	if err != nil {
		log.Printf("FoldersHandler: Failed to get IMAP client: %v", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "i/o timeout") {
			http.Error(w, "Connection to IMAP server timed out. Please double-check your server hostname in your Settings and try again.", http.StatusServiceUnavailable)
		} else {
			http.Error(w, "Failed to connect to IMAP server", http.StatusInternalServerError)
		}
		return nil, false
	}
	return client, true
}

// listFoldersWithRetry lists folders with automatic retry on connection errors.
func (h *FoldersHandler) listFoldersWithRetry(w http.ResponseWriter, userID string, client imap.IMAPClient, settings *models.UserSettings, imapPassword string) ([]*models.Folder, bool) {
	folders, err := client.ListFolders()
	if err != nil {
		return h.handleListFoldersError(w, userID, err, settings, imapPassword)
	}
	return folders, true
}

// handleListFoldersError handles errors from ListFolders, including retry logic.
func (h *FoldersHandler) handleListFoldersError(w http.ResponseWriter, userID string, err error, settings *models.UserSettings, imapPassword string) ([]*models.Folder, bool) {
	log.Printf("FoldersHandler: Failed to list folders: %v", err)
	errMsg := err.Error()

	if strings.Contains(errMsg, "SPECIAL-USE") {
		http.Error(w, "Your IMAP server doesn't support the SPECIAL-USE extension (RFC 6154), which is required for V-Mail to identify folder types. Please contact your email provider or use a different IMAP server.", http.StatusBadRequest)
		return nil, false
	}

	if h.isBrokenConnectionError(errMsg) {
		return h.retryListFolders(w, userID, settings, imapPassword)
	}

	http.Error(w, "Failed to list folders", http.StatusInternalServerError)
	return nil, false
}

// isBrokenConnectionError checks if the error indicates a broken connection.
func (h *FoldersHandler) isBrokenConnectionError(errMsg string) bool {
	return strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "EOF")
}

// retryListFolders retries listing folders after removing the broken connection.
func (h *FoldersHandler) retryListFolders(w http.ResponseWriter, userID string, settings *models.UserSettings, imapPassword string) ([]*models.Folder, bool) {
	h.imapPool.RemoveClient(userID)

	client, retryErr := h.imapPool.GetClient(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword)
	if retryErr != nil {
		log.Printf("FoldersHandler: Failed to get IMAP client on retry: %v", retryErr)
		http.Error(w, "Failed to connect to IMAP server", http.StatusInternalServerError)
		return nil, false
	}

	folders, err := client.ListFolders()
	if err != nil {
		log.Printf("FoldersHandler: Failed to list folders on retry: %v", err)
		if strings.Contains(err.Error(), "SPECIAL-USE") {
			http.Error(w, "Your IMAP server doesn't support the SPECIAL-USE extension (RFC 6154), which is required for V-Mail to identify folder types. Please contact your email provider or use a different IMAP server.", http.StatusBadRequest)
			return nil, false
		}
		http.Error(w, "Failed to list folders", http.StatusInternalServerError)
		return nil, false
	}

	return folders, true
}

// writeFoldersResponse writes the folders response as JSON.
// Uses a buffered approach to prevent partial writes if JSON encoding fails.
func (h *FoldersHandler) writeFoldersResponse(w http.ResponseWriter, folders []*models.Folder) {
	sortFoldersByRole(folders)

	folderValues := make([]models.Folder, len(folders))
	for i, f := range folders {
		folderValues[i] = *f
	}

	// Encode to buffer first to prevent partial writes
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(folderValues); err != nil {
		log.Printf("FoldersHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Only write headers and body if encoding succeeded
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Printf("FoldersHandler: Failed to write response: %v", err)
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
