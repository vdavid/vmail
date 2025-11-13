package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/models"
)

// ErrUserSettingsNotFound is returned when user settings cannot be found.
var ErrUserSettingsNotFound = errors.New("user settings not found")

// GetOrCreateUser returns the user's id for the given email.
// If no user exists with that email, it creates a new one.
func GetOrCreateUser(ctx context.Context, pool *pgxpool.Pool, email string) (string, error) {
	var userID string

	err := pool.QueryRow(ctx, `
		INSERT INTO users (email)
		VALUES ($1)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id
	`, email).Scan(&userID)

	if err != nil {
		return "", fmt.Errorf("failed to get or create user: %w", err)
	}

	return userID, nil
}

// UserSettingsExist returns true if the user settings exist.
func UserSettingsExist(ctx context.Context, pool *pgxpool.Pool, userID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM user_settings WHERE user_id = $1)
	`, userID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check user settings existence: %w", err)
	}

	return exists, nil
}

// GetUserSettings returns the user settings for the given user.
func GetUserSettings(ctx context.Context, pool *pgxpool.Pool, userID string) (*models.UserSettings, error) {
	var settings models.UserSettings

	err := pool.QueryRow(ctx, `
		SELECT 
			user_id,
			undo_send_delay_seconds,
			pagination_threads_per_page,
			imap_server_hostname,
			imap_username,
			encrypted_imap_password,
			smtp_server_hostname,
			smtp_username,
			encrypted_smtp_password,
			created_at,
			updated_at
		FROM user_settings
		WHERE user_id = $1
	`, userID).Scan(
		&settings.UserID,
		&settings.UndoSendDelaySeconds,
		&settings.PaginationThreadsPerPage,
		&settings.IMAPServerHostname,
		&settings.IMAPUsername,
		&settings.EncryptedIMAPPassword,
		&settings.SMTPServerHostname,
		&settings.SMTPUsername,
		&settings.EncryptedSMTPPassword,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserSettingsNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	return &settings, nil
}

// SaveUserSettings saves the user settings for the given user.
func SaveUserSettings(ctx context.Context, pool *pgxpool.Pool, settings *models.UserSettings) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO user_settings (
			user_id,
			undo_send_delay_seconds,
			pagination_threads_per_page,
			imap_server_hostname,
			imap_username,
			encrypted_imap_password,
			smtp_server_hostname,
			smtp_username,
			encrypted_smtp_password
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO UPDATE SET
			undo_send_delay_seconds = EXCLUDED.undo_send_delay_seconds,
			pagination_threads_per_page = EXCLUDED.pagination_threads_per_page,
			imap_server_hostname = EXCLUDED.imap_server_hostname,
			imap_username = EXCLUDED.imap_username,
			encrypted_imap_password = EXCLUDED.encrypted_imap_password,
			smtp_server_hostname = EXCLUDED.smtp_server_hostname,
			smtp_username = EXCLUDED.smtp_username,
			encrypted_smtp_password = EXCLUDED.encrypted_smtp_password,
			updated_at = NOW()
	`,
		settings.UserID,
		settings.UndoSendDelaySeconds,
		settings.PaginationThreadsPerPage,
		settings.IMAPServerHostname,
		settings.IMAPUsername,
		settings.EncryptedIMAPPassword,
		settings.SMTPServerHostname,
		settings.SMTPUsername,
		settings.EncryptedSMTPPassword,
	)

	if err != nil {
		return fmt.Errorf("failed to save user settings: %w", err)
	}

	return nil
}
