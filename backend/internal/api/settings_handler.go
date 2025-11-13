package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
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

	userID, ok := GetUserIDFromContext(ctx, w, h.pool)
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
	}

	// Encode to buffer first to prevent partial writes
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(response); err != nil {
		log.Printf("SettingsHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Only write headers and body if encoding succeeded
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Printf("SettingsHandler: Failed to write response: %v", err)
	}
}

// PostSettings saves or updates the user settings for the current user.
func (h *SettingsHandler) PostSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx, w, h.pool)
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

	// Get existing settings to preserve passwords if not provided in the request.
	// This allows users to update other settings without re-entering passwords.
	existingSettings, err := db.GetUserSettings(ctx, h.pool, userID)
	var encryptedIMAPPassword []byte
	var encryptedSMTPPassword []byte

	if err != nil && !errors.Is(err, db.ErrUserSettingsNotFound) {
		log.Printf("SettingsHandler: Failed to get existing settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Handle IMAP password: use existing if not provided, encrypt new one if provided.
	// For initial setup (no existing settings), password is required.
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

	// Handle SMTP password: use existing if not provided, encrypt new one if provided.
	// For initial setup (no existing settings), password is required.
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
	}

	if err := db.SaveUserSettings(ctx, h.pool, settings); err != nil {
		log.Printf("SettingsHandler: Failed to save settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Encode success response to buffer first to prevent partial writes
	successResponse := struct {
		Success bool `json:"success"`
	}{Success: true}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(successResponse); err != nil {
		log.Printf("SettingsHandler: Failed to encode response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Only write headers and body if encoding succeeded
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Printf("SettingsHandler: Failed to write response: %v", err)
	}
}

// validateSettingsRequest validates the user settings request, ensuring all required
// fields are present. Note that passwords are optional on update (they can be empty
// to preserve existing passwords), but are required for initial setup.
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
