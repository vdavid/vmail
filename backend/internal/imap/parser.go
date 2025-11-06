package imap

import (
	"fmt"
	"io"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/jhillyerd/enmime"
	"github.com/vdavid/vmail/backend/internal/models"
)

// ParseMessage converts an IMAP message to our Message model.
func ParseMessage(imapMsg *imap.Message, threadID, userID, folderName string) (*models.Message, error) {
	if imapMsg == nil {
		return nil, fmt.Errorf("imap message is nil")
	}

	// Check flags
	isRead := false
	isStarred := false
	for _, flag := range imapMsg.Flags {
		if flag == imap.SeenFlag {
			isRead = true
		}
		if flag == imap.FlaggedFlag {
			isStarred = true
		}
	}

	msg := &models.Message{
		ThreadID:       threadID,
		UserID:         userID,
		IMAPUID:        int64(imapMsg.Uid),
		IMAPFolderName: folderName,
		IsRead:         isRead,
		IsStarred:      isStarred,
	}

	if imapMsg.Envelope != nil {
		if len(imapMsg.Envelope.From) > 0 {
			msg.FromAddress = formatAddress(imapMsg.Envelope.From[0])
		}

		msg.ToAddresses = formatAddressList(imapMsg.Envelope.To)
		msg.CCAddresses = formatAddressList(imapMsg.Envelope.Cc)
		msg.Subject = imapMsg.Envelope.Subject
		if !imapMsg.Envelope.Date.IsZero() {
			msg.SentAt = &imapMsg.Envelope.Date
		}
	}

	if imapMsg.Envelope != nil && len(imapMsg.Envelope.MessageId) > 0 {
		msg.MessageIDHeader = imapMsg.Envelope.MessageId
	}

	// Parse body if available
	if imapMsg.Body != nil && imapMsg.BodyStructure != nil {
		section := &imap.BodySectionName{}
		bodyReader := imapMsg.GetBody(section)
		if bodyReader != nil {
			if err := parseBody(bodyReader, msg); err != nil {
				// Log error but don't fail - we still have headers
				_ = err
			}
		}
	}

	return msg, nil
}

// parseBody parses the email body using enmime.
func parseBody(bodyReader io.Reader, msg *models.Message) error {
	envelope, err := enmime.ReadEnvelope(bodyReader)
	if err != nil {
		return fmt.Errorf("failed to parse email body: %w", err)
	}

	// Get HTML body
	htmlBody := envelope.HTML
	if htmlBody == "" {
		// If no HTML, convert text to HTML
		htmlBody = strings.ReplaceAll(envelope.Text, "\n", "<br>")
	}
	msg.UnsafeBodyHTML = htmlBody
	msg.BodyText = envelope.Text

	// Parse attachments
	for _, part := range envelope.Attachments {
		attachment := &models.Attachment{
			Filename:  part.FileName,
			MimeType:  part.ContentType,
			SizeBytes: int64(len(part.Content)),
			IsInline:  false,
		}

		if part.ContentID != "" {
			attachment.ContentID = part.ContentID
			attachment.IsInline = true
		}

		msg.Attachments = append(msg.Attachments, *attachment)
	}

	return nil
}

// formatAddress formats an IMAP address to a string.
func formatAddress(address *imap.Address) string {
	if address == nil {
		return ""
	}

	if address.MailboxName == "" && address.HostName == "" {
		return ""
	}

	if address.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>", address.PersonalName, address.MailboxName, address.HostName)
	}

	return fmt.Sprintf("%s@%s", address.MailboxName, address.HostName)
}

// formatAddressList formats a list of IMAP addresses.
func formatAddressList(addresses []*imap.Address) []string {
	result := make([]string, 0, len(addresses))
	for _, address := range addresses {
		formatted := formatAddress(address)
		if formatted != "" {
			result = append(result, formatted)
		}
	}
	return result
}

// ExtractStableThreadID extracts the stable thread ID from a message.
// This uses the Message-ID header of the root message.
func ExtractStableThreadID(envelope *imap.Envelope) string {
	if envelope == nil || len(envelope.MessageId) == 0 {
		return ""
	}
	return envelope.MessageId
}
