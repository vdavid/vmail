package imap

import (
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/models"
)

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  *imap.Address
		expected string
	}{
		{
			name: "formats address with personal name",
			address: &imap.Address{
				PersonalName: "John Doe",
				MailboxName:  "john",
				HostName:     "example.com",
			},
			expected: "John Doe <john@example.com>",
		},
		{
			name: "formats address without personal name",
			address: &imap.Address{
				MailboxName: "jane",
				HostName:    "example.com",
			},
			expected: "jane@example.com",
		},
		{
			name:     "returns empty string for nil address",
			address:  nil,
			expected: "",
		},
		{
			name:     "returns empty string for empty address",
			address:  &imap.Address{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAddress(tt.address)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAddressList(t *testing.T) {
	tests := []struct {
		name      string
		addresses []*imap.Address
		expected  []string
	}{
		{
			name: "formats list of addresses",
			addresses: []*imap.Address{
				{
					MailboxName: "user1",
					HostName:    "example.com",
				},
				{
					PersonalName: "User Two",
					MailboxName:  "user2",
					HostName:     "example.com",
				},
			},
			expected: []string{"user1@example.com", "User Two <user2@example.com>"},
		},
		{
			name:      "returns empty list for empty input",
			addresses: []*imap.Address{},
			expected:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAddressList(tt.addresses)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractStableThreadID(t *testing.T) {
	tests := []struct {
		name     string
		envelope *imap.Envelope
		expected string
	}{
		{
			name: "extracts Message-ID from envelope",
			envelope: &imap.Envelope{
				MessageId: "<test-message-id@example.com>",
			},
			expected: "<test-message-id@example.com>",
		},
		{
			name:     "returns empty string for nil envelope",
			envelope: nil,
			expected: "",
		},
		{
			name:     "returns empty string when Message-ID is missing",
			envelope: &imap.Envelope{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractStableThreadID(tt.envelope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMessage(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		imapMsg     *imap.Message
		threadID    string
		userID      string
		folderName  string
		expectError bool
		checkResult func(*testing.T, *models.Message)
	}{
		{
			name: "parses message with envelope",
			imapMsg: &imap.Message{
				Uid:   100,
				Flags: []string{imap.SeenFlag, imap.FlaggedFlag},
				Envelope: &imap.Envelope{
					MessageId: "<msg-123@example.com>",
					From: []*imap.Address{
						{
							PersonalName: "Sender",
							MailboxName:  "sender",
							HostName:     "example.com",
						},
					},
					To: []*imap.Address{
						{
							MailboxName: "recipient",
							HostName:    "example.com",
						},
					},
					Subject: "Test Subject",
					Date:    now,
				},
			},
			threadID:   "thread-id",
			userID:     "user-id",
			folderName: "INBOX",
			checkResult: func(t *testing.T, msg *models.Message) {
				assert.Equal(t, int64(100), msg.IMAPUID)
				assert.Equal(t, "INBOX", msg.IMAPFolderName)
				assert.True(t, msg.IsRead)
				assert.True(t, msg.IsStarred)
				assert.Equal(t, "<msg-123@example.com>", msg.MessageIDHeader)
				assert.Contains(t, msg.FromAddress, "Sender")
				assert.Len(t, msg.ToAddresses, 1)
				assert.Equal(t, "Test Subject", msg.Subject)
				assert.NotNil(t, msg.SentAt)
				assert.True(t, msg.SentAt.Equal(now))
			},
		},
		{
			name:        "handles nil message",
			imapMsg:     nil,
			threadID:    "thread-id",
			userID:      "user-id",
			folderName:  "INBOX",
			expectError: true,
		},
		{
			name: "handles message without envelope",
			imapMsg: &imap.Message{
				Uid:   200,
				Flags: []string{},
			},
			threadID:   "thread-id",
			userID:     "user-id",
			folderName: "INBOX",
			checkResult: func(t *testing.T, msg *models.Message) {
				assert.Equal(t, int64(200), msg.IMAPUID)
				assert.False(t, msg.IsRead)
			},
		},
		{
			name: "handles message without Message-ID",
			imapMsg: &imap.Message{
				Uid:   300,
				Flags: []string{},
				Envelope: &imap.Envelope{
					Subject: "Test Subject",
				},
			},
			threadID:   "thread-id",
			userID:     "user-id",
			folderName: "INBOX",
			checkResult: func(t *testing.T, msg *models.Message) {
				assert.Empty(t, msg.MessageIDHeader)
				assert.Equal(t, "Test Subject", msg.Subject)
			},
		},
		{
			name: "handles message with empty body",
			imapMsg: &imap.Message{
				Uid:   400,
				Flags: []string{},
				Envelope: &imap.Envelope{
					MessageId: "<empty-body@example.com>",
				},
			},
			threadID:   "thread-id",
			userID:     "user-id",
			folderName: "INBOX",
			checkResult: func(t *testing.T, msg *models.Message) {
				assert.Empty(t, msg.UnsafeBodyHTML)
				assert.Empty(t, msg.BodyText)
			},
		},
		{
			name: "handles body parsing errors gracefully",
			imapMsg: &imap.Message{
				Uid:   500,
				Flags: []string{},
				Envelope: &imap.Envelope{
					MessageId: "<invalid-body@example.com>",
					Subject:   "Test Subject",
				},
				BodyStructure: &imap.BodyStructure{
					MIMEType:    "text",
					MIMESubType: "plain",
				},
			},
			threadID:   "thread-id",
			userID:     "user-id",
			folderName: "INBOX",
			checkResult: func(t *testing.T, msg *models.Message) {
				assert.Equal(t, "Test Subject", msg.Subject)
				assert.Equal(t, "<invalid-body@example.com>", msg.MessageIDHeader)
			},
		},
		{
			name: "handles message with attachments",
			imapMsg: &imap.Message{
				Uid:   600,
				Flags: []string{},
				Envelope: &imap.Envelope{
					MessageId: "<with-attachments@example.com>",
					Subject:   "Test with Attachments",
				},
				BodyStructure: &imap.BodyStructure{
					MIMEType:    "multipart",
					MIMESubType: "mixed",
					Parts: []*imap.BodyStructure{
						{
							MIMEType:    "text",
							MIMESubType: "plain",
						},
						{
							MIMEType:    "application",
							MIMESubType: "pdf",
							Disposition: "attachment",
							DispositionParams: map[string]string{
								"filename": "test.pdf",
							},
						},
					},
				},
			},
			threadID:   "thread-id",
			userID:     "user-id",
			folderName: "INBOX",
			checkResult: func(t *testing.T, msg *models.Message) {
				assert.Equal(t, "Test with Attachments", msg.Subject)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseMessage(tt.imapMsg, tt.threadID, tt.userID, tt.folderName)
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.checkResult != nil {
				tt.checkResult(t, msg)
			}
		})
	}
}
