package models

import "time"

// Folder represents an IMAP folder.
type Folder struct {
	Name string `json:"name"`
}

// Thread represents an email thread containing multiple messages.
// A thread is a folder-agnostic container that groups related messages together.
// The StableThreadID is the Message-ID header of the root message, which allows
// us to group messages from different folders (e.g., 'INBOX' and 'Sent') into a single thread.
type Thread struct {
	ID             string    `json:"id"`
	StableThreadID string    `json:"stable_thread_id"`
	Subject        string    `json:"subject"`
	UserID         string    `json:"user_id"`
	Messages       []Message `json:"messages,omitempty"`
}

// Message represents a single email message.
// Messages are cached in the database for fast UI rendering.
// The user_id field is denormalized for performance (avoids JOINs when querying by user).
// The imap_uid is only unique within a specific imap_folder_name.
type Message struct {
	ID              string       `json:"id"`
	ThreadID        string       `json:"thread_id"`
	UserID          string       `json:"user_id"`
	IMAPUID         int64        `json:"imap_uid"`
	IMAPFolderName  string       `json:"imap_folder_name"`
	MessageIDHeader string       `json:"message_id_header"`
	FromAddress     string       `json:"from_address"`
	ToAddresses     []string     `json:"to_addresses"`
	CCAddresses     []string     `json:"cc_addresses"`
	SentAt          *time.Time   `json:"sent_at"`
	Subject         string       `json:"subject"`
	UnsafeBodyHTML  string       `json:"unsafe_body_html"`
	BodyText        string       `json:"body_text"`
	IsRead          bool         `json:"is_read"`
	IsStarred       bool         `json:"is_starred"`
	Attachments     []Attachment `json:"attachments,omitempty"`
}

// Attachment represents an email attachment.
// If IsInline is true, the attachment is meant to be shown inside the email body
// (e.g., a signature image). The ContentID is used to match inline attachments
// to <img src="cid:..."> tags in the email HTML.
type Attachment struct {
	ID        string `json:"id"`
	MessageID string `json:"message_id"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	IsInline  bool   `json:"is_inline"`
	ContentID string `json:"content_id,omitempty"`
}
