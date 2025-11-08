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

type FoldersHandler struct {
	pool      *pgxpool.Pool
	encryptor *crypto.Encryptor
	imapPool  imap.IMAPPool
}

func NewFoldersHandler(pool *pgxpool.Pool, encryptor *crypto.Encryptor, imapPool imap.IMAPPool) *FoldersHandler {
	return &FoldersHandler{
		pool:      pool,
		encryptor: encryptor,
		imapPool:  imapPool,
	}
}

func (h *FoldersHandler) GetFolders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := h.getUserIDFromContext(ctx, w)
	if !ok {
		return
	}

	// Get user settings
	settings, err := db.GetUserSettings(ctx, h.pool, userID)
	if err != nil {
		if errors.Is(err, db.ErrUserSettingsNotFound) {
			http.Error(w, "User settings not found", http.StatusNotFound)
			return
		}
		log.Printf("FoldersHandler: Failed to get user settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Decrypt password
	imapPassword, err := h.encryptor.Decrypt(settings.EncryptedIMAPPassword)
	if err != nil {
		log.Printf("FoldersHandler: Failed to decrypt IMAP password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get IMAP client
	client, err := h.imapPool.GetClient(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword)
	if err != nil {
		log.Printf("FoldersHandler: Failed to get IMAP client: %v", err)
		// Check if it's a timeout or connection error
		errMsg := err.Error()
		if strings.Contains(errMsg, "i/o timeout") {
			http.Error(w, "Connection to IMAP server timed out. Please double-check your server hostname in your Settings and try again.", http.StatusServiceUnavailable)
		} else {
			http.Error(w, "Failed to connect to IMAP server", http.StatusInternalServerError)
		}
		return
	}

	// List folders using the interface method
	folderNames, err := client.ListFolders()
	if err != nil {
		log.Printf("FoldersHandler: Failed to list folders: %v", err)

		// Check if it's a broken connection error
		errMsg := err.Error()
		if strings.Contains(errMsg, "broken pipe") || strings.Contains(errMsg, "connection reset") || strings.Contains(errMsg, "EOF") {
			// Remove the dead connection from the pool and retry once
			h.imapPool.RemoveClient(userID)

			// Retry with a fresh connection
			client, retryErr := h.imapPool.GetClient(userID, settings.IMAPServerHostname, settings.IMAPUsername, imapPassword)
			if retryErr != nil {
				log.Printf("FoldersHandler: Failed to get IMAP client on retry: %v", retryErr)
				http.Error(w, "Failed to connect to IMAP server", http.StatusInternalServerError)
				return
			}

			folderNames, err = client.ListFolders()
			if err != nil {
				log.Printf("FoldersHandler: Failed to list folders on retry: %v", err)
				http.Error(w, "Failed to list folders", http.StatusInternalServerError)
				return
			}
			// Success on retry, continue below
		} else {
			http.Error(w, "Failed to list folders", http.StatusInternalServerError)
			return
		}
	}

	// Convert to response format
	folders := make([]models.Folder, 0, len(folderNames))
	for _, name := range folderNames {
		folders = append(folders, models.Folder{Name: name})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(folders); err != nil {
		log.Printf("FoldersHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *FoldersHandler) getUserIDFromContext(ctx context.Context, w http.ResponseWriter) (string, bool) {
	email, ok := auth.GetUserEmailFromContext(ctx)
	if !ok {
		log.Println("FoldersHandler: No user email in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}

	userID, err := db.GetOrCreateUser(ctx, h.pool, email)
	if err != nil {
		log.Printf("FoldersHandler: Failed to get/create user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return "", false
	}

	return userID, true
}
