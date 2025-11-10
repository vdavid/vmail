package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/models"
)

// SettingsHandler handles user settings-related API requests.
type SettingsHandler struct {
	pool      *pgxpool.Pool
	encryptor *crypto.Encryptor
}

// NewSettingsHandler creates a new SettingsHandler instance.
func NewSettingsHandler(pool *pgxpool.Pool, encryptor *crypto.Encryptor) *SettingsHandler {
	return &SettingsHandler{
		pool:      pool,
		encryptor: encryptor,
	}
}

// GetSettings returns the user settings for the current user.
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := h.getUserIDFromContext(ctx, w)
	if !ok {
		return
	}

	settings, err := db.GetUserSettings(ctx, h.pool, userID)
	if errors.Is(err, db.ErrUserSettingsNotFound) {
		http.Error(w, "Settings not found for this user", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("SettingsHandler: Failed to get settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := models.UserSettingsResponse{
		UndoSendDelaySeconds:     settings.UndoSendDelaySeconds,
		PaginationThreadsPerPage: settings.PaginationThreadsPerPage,
		IMAPServerHostname:       settings.IMAPServerHostname,
		IMAPUsername:             settings.IMAPUsername,
		IMAPPasswordSet:          len(settings.EncryptedIMAPPassword) > 0,
		SMTPServerHostname:       settings.SMTPServerHostname,
		SMTPUsername:             settings.SMTPUsername,
		SMTPPasswordSet:          len(settings.EncryptedSMTPPassword) > 0,
		ArchiveFolderName:        settings.ArchiveFolderName,
		SentFolderName:           settings.SentFolderName,
		DraftsFolderName:         settings.DraftsFolderName,
		TrashFolderName:          settings.TrashFolderName,
		SpamFolderName:           settings.SpamFolderName,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("SettingsHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// PostSettings saves or updates the user settings for the current user.
func (h *SettingsHandler) PostSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := h.getUserIDFromContext(ctx, w)
	if !ok {
		return
	}

	var req models.UserSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("SettingsHandler: Failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validateSettingsRequest(&req); err != nil {
		log.Printf("SettingsHandler: Validation failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get existing settings to preserve passwords if not provided
	existingSettings, err := db.GetUserSettings(ctx, h.pool, userID)
	var encryptedIMAPPassword []byte
	var encryptedSMTPPassword []byte

	if err != nil && !errors.Is(err, db.ErrUserSettingsNotFound) {
		log.Printf("SettingsHandler: Failed to get existing settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Handle IMAP password: use existing if not provided, encrypt new one if provided
	if req.IMAPPassword == "" {
		if existingSettings != nil {
			encryptedIMAPPassword = existingSettings.EncryptedIMAPPassword
		} else {
			// First time setup requires password
			http.Error(w, "IMAP password is required for initial setup", http.StatusBadRequest)
			return
		}
	} else {
		var err error
		encryptedIMAPPassword, err = h.encryptor.Encrypt(req.IMAPPassword)
		if err != nil {
			log.Printf("SettingsHandler: Failed to encrypt IMAP password: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	// Handle SMTP password: use existing if not provided, encrypt new one if provided
	if req.SMTPPassword == "" {
		if existingSettings != nil {
			encryptedSMTPPassword = existingSettings.EncryptedSMTPPassword
		} else {
			// First time setup requires password
			http.Error(w, "SMTP password is required for initial setup", http.StatusBadRequest)
			return
		}
	} else {
		var err error
		encryptedSMTPPassword, err = h.encryptor.Encrypt(req.SMTPPassword)
		if err != nil {
			log.Printf("SettingsHandler: Failed to encrypt SMTP password: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	settings := &models.UserSettings{
		UserID:                   userID,
		UndoSendDelaySeconds:     req.UndoSendDelaySeconds,
		PaginationThreadsPerPage: req.PaginationThreadsPerPage,
		IMAPServerHostname:       req.IMAPServerHostname,
		IMAPUsername:             req.IMAPUsername,
		EncryptedIMAPPassword:    encryptedIMAPPassword,
		SMTPServerHostname:       req.SMTPServerHostname,
		SMTPUsername:             req.SMTPUsername,
		EncryptedSMTPPassword:    encryptedSMTPPassword,
		ArchiveFolderName:        req.ArchiveFolderName,
		SentFolderName:           req.SentFolderName,
		DraftsFolderName:         req.DraftsFolderName,
		TrashFolderName:          req.TrashFolderName,
		SpamFolderName:           req.SpamFolderName,
	}

	if err := db.SaveUserSettings(ctx, h.pool, settings); err != nil {
		log.Printf("SettingsHandler: Failed to save settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(`{"success": true}`))
	if err != nil {
		return
	}
}

func (h *SettingsHandler) validateSettingsRequest(req *models.UserSettingsRequest) error {
	if req.IMAPServerHostname == "" {
		return errors.New("IMAP server hostname is required")
	}
	if req.IMAPUsername == "" {
		return errors.New("IMAP username is required")
	}
	// Password validation removed - passwords are optional on update
	if req.SMTPServerHostname == "" {
		return errors.New("SMTP server hostname is required")
	}
	if req.SMTPUsername == "" {
		return errors.New("SMTP username is required")
	}
	// Password validation removed - passwords are optional on update
	return nil
}

// getUserIDFromContext extracts the user's email from context, resolves/creates the DB user,
// and writes appropriate HTTP errors when it fails. Returns (userID, true) on success.
func (h *SettingsHandler) getUserIDFromContext(ctx context.Context, w http.ResponseWriter) (string, bool) {
	email, ok := auth.GetUserEmailFromContext(ctx)
	if !ok {
		log.Println("SettingsHandler: No user email in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}

	userID, err := db.GetOrCreateUser(ctx, h.pool, email)
	if err != nil {
		log.Printf("SettingsHandler: Failed to get/create user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return "", false
	}

	return userID, true
}
