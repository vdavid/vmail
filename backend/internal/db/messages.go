package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/models"
)

// ErrMessageNotFound is returned when a requested message cannot be found.
var ErrMessageNotFound = errors.New("message not found")

// SaveMessage saves or updates a message in the database.
func SaveMessage(ctx context.Context, pool *pgxpool.Pool, message *models.Message) error {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO messages (
			thread_id,
			user_id,
			imap_uid,
			imap_folder_name,
			message_id_header,
			from_address,
			to_addresses,
			cc_addresses,
			sent_at,
			subject,
			unsafe_body_html,
			body_text,
			is_read,
			is_starred
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (user_id, imap_folder_name, imap_uid) DO UPDATE SET
			thread_id = EXCLUDED.thread_id,
			message_id_header = EXCLUDED.message_id_header,
			from_address = EXCLUDED.from_address,
			to_addresses = EXCLUDED.to_addresses,
			cc_addresses = EXCLUDED.cc_addresses,
			sent_at = EXCLUDED.sent_at,
			subject = EXCLUDED.subject,
			unsafe_body_html = COALESCE(EXCLUDED.unsafe_body_html, messages.unsafe_body_html),
			body_text = COALESCE(EXCLUDED.body_text, messages.body_text),
			is_read = EXCLUDED.is_read,
			is_starred = EXCLUDED.is_starred
		RETURNING id
    `,
		message.ThreadID,
		message.UserID,
		message.IMAPUID,
		message.IMAPFolderName,
		message.MessageIDHeader,
		message.FromAddress,
		message.ToAddresses,
		message.CCAddresses,
		message.SentAt,
		message.Subject,
		message.UnsafeBodyHTML,
		message.BodyText,
		message.IsRead,
		message.IsStarred,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Populate the ID if we inserted a new row
	if id != "" {
		message.ID = id
	}

	return nil
}

// GetMessagesForThread returns all messages for a thread.
func GetMessagesForThread(ctx context.Context, pool *pgxpool.Pool, threadID string) ([]*models.Message, error) {
	rows, err := pool.Query(ctx, `
		SELECT 
			id,
			thread_id,
			user_id,
			imap_uid,
			imap_folder_name,
			message_id_header,
			from_address,
			to_addresses,
			cc_addresses,
			sent_at,
			subject,
			unsafe_body_html,
			body_text,
			is_read,
			is_starred
		FROM messages
		WHERE thread_id = $1
		ORDER BY sent_at NULLS LAST
	`, threadID)

	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(
			&msg.ID,
			&msg.ThreadID,
			&msg.UserID,
			&msg.IMAPUID,
			&msg.IMAPFolderName,
			&msg.MessageIDHeader,
			&msg.FromAddress,
			&msg.ToAddresses,
			&msg.CCAddresses,
			&msg.SentAt,
			&msg.Subject,
			&msg.UnsafeBodyHTML,
			&msg.BodyText,
			&msg.IsRead,
			&msg.IsStarred,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// GetMessageByUID returns a message by its IMAP UID and folder.
func GetMessageByUID(ctx context.Context, pool *pgxpool.Pool, userID, folderName string, imapUID int64) (*models.Message, error) {
	var msg models.Message

	err := pool.QueryRow(ctx, `
		SELECT 
			id,
			thread_id,
			user_id,
			imap_uid,
			imap_folder_name,
			message_id_header,
			from_address,
			to_addresses,
			cc_addresses,
			sent_at,
			subject,
			unsafe_body_html,
			body_text,
			is_read,
			is_starred
		FROM messages
		WHERE user_id = $1 AND imap_folder_name = $2 AND imap_uid = $3
	`, userID, folderName, imapUID).Scan(
		&msg.ID,
		&msg.ThreadID,
		&msg.UserID,
		&msg.IMAPUID,
		&msg.IMAPFolderName,
		&msg.MessageIDHeader,
		&msg.FromAddress,
		&msg.ToAddresses,
		&msg.CCAddresses,
		&msg.SentAt,
		&msg.Subject,
		&msg.UnsafeBodyHTML,
		&msg.BodyText,
		&msg.IsRead,
		&msg.IsStarred,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMessageNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &msg, nil
}

// SaveAttachment saves an attachment to the database.
func SaveAttachment(ctx context.Context, pool *pgxpool.Pool, attachment *models.Attachment) error {
	var attachmentID string

	err := pool.QueryRow(ctx, `
		INSERT INTO attachments (message_id, filename, mime_type, size_bytes, is_inline, content_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, attachment.MessageID, attachment.Filename, attachment.MimeType, attachment.SizeBytes, attachment.IsInline, attachment.ContentID).Scan(&attachmentID)

	if err != nil {
		return fmt.Errorf("failed to save attachment: %w", err)
	}

	attachment.ID = attachmentID
	return nil
}

// GetAttachmentsForMessage returns all attachments for a message.
func GetAttachmentsForMessage(ctx context.Context, pool *pgxpool.Pool, messageID string) ([]*models.Attachment, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, message_id, filename, mime_type, size_bytes, is_inline, content_id
		FROM attachments
		WHERE message_id = $1
	`, messageID)

	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*models.Attachment
	for rows.Next() {
		var att models.Attachment
		if err := rows.Scan(
			&att.ID,
			&att.MessageID,
			&att.Filename,
			&att.MimeType,
			&att.SizeBytes,
			&att.IsInline,
			&att.ContentID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, &att)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attachments: %w", err)
	}

	return attachments, nil
}

// GetAttachmentsForMessages returns all attachments for multiple messages in a single query.
// Returns a map from message ID to a slice of attachments.
func GetAttachmentsForMessages(ctx context.Context, pool *pgxpool.Pool, messageIDs []string) (map[string][]*models.Attachment, error) {
	if len(messageIDs) == 0 {
		return make(map[string][]*models.Attachment), nil
	}

	rows, err := pool.Query(ctx, `
		SELECT id, message_id, filename, mime_type, size_bytes, is_inline, content_id
		FROM attachments
		WHERE message_id = ANY($1)
		ORDER BY message_id
	`, messageIDs)

	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	defer rows.Close()

	attachmentsMap := make(map[string][]*models.Attachment)
	for rows.Next() {
		var att models.Attachment
		if err := rows.Scan(
			&att.ID,
			&att.MessageID,
			&att.Filename,
			&att.MimeType,
			&att.SizeBytes,
			&att.IsInline,
			&att.ContentID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachmentsMap[att.MessageID] = append(attachmentsMap[att.MessageID], &att)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attachments: %w", err)
	}

	return attachmentsMap, nil
}
