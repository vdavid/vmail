package models

import (
	"time"
)

// User represents a V-Mail user.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserSettings holds user-specific application settings and encrypted credentials.
type UserSettings struct {
	UserID                   string    `json:"user_id"`
	UndoSendDelaySeconds     int       `json:"undo_send_delay_seconds"`
	PaginationThreadsPerPage int       `json:"pagination_threads_per_page"`
	IMAPServerHostname       string    `json:"imap_server_hostname"`
	IMAPUsername             string    `json:"imap_username"`
	EncryptedIMAPPassword    []byte    `json:"-"`
	SMTPServerHostname       string    `json:"smtp_server_hostname"`
	SMTPUsername             string    `json:"smtp_username"`
	EncryptedSMTPPassword    []byte    `json:"-"`
	ArchiveFolderName        string    `json:"archive_folder_name"`
	SentFolderName           string    `json:"sent_folder_name"`
	DraftsFolderName         string    `json:"drafts_folder_name"`
	TrashFolderName          string    `json:"trash_folder_name"`
	SpamFolderName           string    `json:"spam_folder_name"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// UserSettingsRequest represents the request payload for saving user settings.
type UserSettingsRequest struct {
	UndoSendDelaySeconds     int    `json:"undo_send_delay_seconds"`
	PaginationThreadsPerPage int    `json:"pagination_threads_per_page"`
	IMAPServerHostname       string `json:"imap_server_hostname"`
	IMAPUsername             string `json:"imap_username"`
	IMAPPassword             string `json:"imap_password"`
	SMTPServerHostname       string `json:"smtp_server_hostname"`
	SMTPUsername             string `json:"smtp_username"`
	SMTPPassword             string `json:"smtp_password"`
	ArchiveFolderName        string `json:"archive_folder_name"`
	SentFolderName           string `json:"sent_folder_name"`
	DraftsFolderName         string `json:"drafts_folder_name"`
	TrashFolderName          string `json:"trash_folder_name"`
	SpamFolderName           string `json:"spam_folder_name"`
}

// UserSettingsResponse represents the response payload for user settings (passwords are never included).
type UserSettingsResponse struct {
	UndoSendDelaySeconds     int    `json:"undo_send_delay_seconds"`
	PaginationThreadsPerPage int    `json:"pagination_threads_per_page"`
	IMAPServerHostname       string `json:"imap_server_hostname"`
	IMAPUsername             string `json:"imap_username"`
	IMAPPasswordSet          bool   `json:"imap_password_set"`
	SMTPServerHostname       string `json:"smtp_server_hostname"`
	SMTPUsername             string `json:"smtp_username"`
	SMTPPasswordSet          bool   `json:"smtp_password_set"`
	ArchiveFolderName        string `json:"archive_folder_name"`
	SentFolderName           string `json:"sent_folder_name"`
	DraftsFolderName         string `json:"drafts_folder_name"`
	TrashFolderName          string `json:"trash_folder_name"`
	SpamFolderName           string `json:"spam_folder_name"`
}

// AuthStatusResponse represents the authentication and setup status of a user.
type AuthStatusResponse struct {
	IsAuthenticated bool `json:"isAuthenticated"`
	IsSetupComplete bool `json:"isSetupComplete"`
}
