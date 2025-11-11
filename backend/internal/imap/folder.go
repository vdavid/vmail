package imap

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/vdavid/vmail/backend/internal/models"
)

// ListFolders lists all folders on the IMAP server with their roles determined by SPECIAL-USE attributes (RFC 6154).
// Returns an error if the server doesn't support SPECIAL-USE extension.
func ListFolders(c *client.Client) ([]*models.Folder, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}

	// Check if server supports SPECIAL-USE extension
	caps, err := c.Capability()
	if err != nil {
		return nil, fmt.Errorf("failed to check server capabilities: %w", err)
	}

	// SPECIAL-USE is required for V-Mail to identify folder roles
	if !caps["SPECIAL-USE"] {
		return nil, fmt.Errorf("IMAP server does not support SPECIAL-USE extension (RFC 6154), which is required for V-Mail to identify folder types")
	}

	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)

	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	var folders []*models.Folder
	for m := range mailboxes {
		role := determineFolderRole(m.Name, m.Attributes)
		folders = append(folders, &models.Folder{
			Name: m.Name,
			Role: role,
		})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return folders, nil
}

// determineFolderRole determines the role of a folder based on its name and SPECIAL-USE attributes.
// INBOX is identified by name (case-insensitive), other roles by SPECIAL-USE attributes.
func determineFolderRole(name string, attributes []string) string {
	// INBOX is always identified by name (case-insensitive), not by SPECIAL-USE attribute
	if strings.EqualFold(name, "INBOX") {
		return "inbox"
	}

	// Check SPECIAL-USE attributes (RFC 6154)
	for _, attr := range attributes {
		switch attr {
		case "\\Sent":
			return "sent"
		case "\\Drafts":
			return "drafts"
		case "\\Junk":
			return "spam"
		case "\\Trash":
			return "trash"
		case "\\Archive":
			return "archive"
		}
	}

	// Default to "other" if no special role is identified
	return "other"
}
