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
// Extracts headers, flags, and body (if available). Body parsing errors are logged but don't fail the parse.
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

	// Parse regular attachments (downloadable files)
	for _, part := range envelope.Attachments {
		attachment := createAttachment(part, false)
		msg.Attachments = append(msg.Attachments, attachment)
	}

	// Parse inline parts (embedded images, etc.)
	for _, part := range envelope.Inlines {
		attachment := createAttachment(part, true)
		msg.Attachments = append(msg.Attachments, attachment)
	}

	// Also check OtherParts for any parts with ContentID (inline images that enmime didn't classify)
	for _, part := range envelope.OtherParts {
		if part.ContentID != "" {
			attachment := createAttachment(part, true)
			msg.Attachments = append(msg.Attachments, attachment)
		}
	}

	return nil
}

// createAttachment creates an Attachment from an enmime Part.
func createAttachment(part *enmime.Part, isInline bool) models.Attachment {
	contentID := strings.Trim(part.ContentID, "<>")
	return models.Attachment{
		Filename:  part.FileName,
		MimeType:  part.ContentType,
		SizeBytes: int64(len(part.Content)),
		IsInline:  isInline || contentID != "",
		ContentID: contentID,
	}
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

// ExtractReferences extracts the References header from an IMAP message.
// The References header contains a space-separated list of Message-IDs
// from the root message to the parent (the message being replied to).
// Returns nil if no References header is found.
func ExtractReferences(imapMsg *imap.Message) []string {
	if imapMsg == nil || imapMsg.Body == nil {
		return nil
	}

	// Find the References header in the Body map
	for key, reader := range imapMsg.Body {
		// Check if this is the References header section
		if strings.Contains(strings.ToUpper(string(key.FetchItem())), "REFERENCES") {
			data, err := io.ReadAll(reader)
			if err != nil {
				return nil
			}
			return parseReferencesHeader(string(data))
		}
	}

	return nil
}

// parseReferencesHeader parses the References header value into a slice of Message-IDs.
func parseReferencesHeader(header string) []string {
	header = strings.TrimSpace(header)
	lowerHeader := strings.ToLower(header)
	if strings.HasPrefix(lowerHeader, "references:") {
		header = strings.TrimSpace(header[11:])
	}

	if header == "" {
		return nil
	}

	parts := strings.Fields(header)
	refs := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, "'\"")
		if part != "" {
			refs = append(refs, part)
		}
	}

	if len(refs) == 0 {
		return nil
	}

	return refs
}
