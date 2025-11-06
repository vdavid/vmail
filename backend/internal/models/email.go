package models

import "time"

type Folder struct {
	Name string `json:"name"`
}

type Thread struct {
	ID             string    `json:"id"`
	StableThreadID string    `json:"stable_thread_id"`
	Subject        string    `json:"subject"`
	UserID         string    `json:"user_id"`
	Messages       []Message `json:"messages,omitempty"`
}

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

type Attachment struct {
	ID        string `json:"id"`
	MessageID string `json:"message_id"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	IsInline  bool   `json:"is_inline"`
	ContentID string `json:"content_id,omitempty"`
}
