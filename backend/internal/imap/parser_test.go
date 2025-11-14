package imap

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

func TestFormatAddress(t *testing.T) {
	t.Run("formats address with personal name", func(t *testing.T) {
		address := &imap.Address{
			PersonalName: "John Doe",
			MailboxName:  "john",
			HostName:     "example.com",
		}

		result := formatAddress(address)
		expected := "John Doe <john@example.com>"
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("formats address without personal name", func(t *testing.T) {
		address := &imap.Address{
			MailboxName: "jane",
			HostName:    "example.com",
		}

		result := formatAddress(address)
		expected := "jane@example.com"
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("returns empty string for nil address", func(t *testing.T) {
		result := formatAddress(nil)
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})

	t.Run("returns empty string for empty address", func(t *testing.T) {
		address := &imap.Address{}
		result := formatAddress(address)
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})
}

func TestFormatAddressList(t *testing.T) {
	t.Run("formats list of addresses", func(t *testing.T) {
		addresses := []*imap.Address{
			{
				MailboxName: "user1",
				HostName:    "example.com",
			},
			{
				PersonalName: "User Two",
				MailboxName:  "user2",
				HostName:     "example.com",
			},
		}

		result := formatAddressList(addresses)
		if len(result) != 2 {
			t.Errorf("Expected 2 addresses, got %d", len(result))
		}
		if result[0] != "user1@example.com" {
			t.Errorf("Expected first address 'user1@example.com', got %s", result[0])
		}
		if result[1] != "User Two <user2@example.com>" {
			t.Errorf("Expected second address 'User Two <user2@example.com>', got %s", result[1])
		}
	})

	t.Run("returns empty list for empty input", func(t *testing.T) {
		result := formatAddressList([]*imap.Address{})
		if len(result) != 0 {
			t.Errorf("Expected empty list, got %d items", len(result))
		}
	})
}

func TestExtractStableThreadID(t *testing.T) {
	t.Run("extracts Message-ID from envelope", func(t *testing.T) {
		envelope := &imap.Envelope{
			MessageId: "<test-message-id@example.com>",
		}

		result := ExtractStableThreadID(envelope)
		if result != "<test-message-id@example.com>" {
			t.Errorf("Expected '<test-message-id@example.com>', got %s", result)
		}
	})

	t.Run("returns empty string for nil envelope", func(t *testing.T) {
		result := ExtractStableThreadID(nil)
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})

	t.Run("returns empty string when Message-ID is missing", func(t *testing.T) {
		envelope := &imap.Envelope{}
		result := ExtractStableThreadID(envelope)
		if result != "" {
			t.Errorf("Expected empty string, got %s", result)
		}
	})
}

func TestParseMessage(t *testing.T) {
	t.Run("parses message with envelope", func(t *testing.T) {
		now := time.Now()
		imapMsg := &imap.Message{
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
		}

		msg, err := ParseMessage(imapMsg, "thread-id", "user-id", "INBOX")
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}

		if msg.IMAPUID != 100 {
			t.Errorf("Expected IMAPUID 100, got %d", msg.IMAPUID)
		}
		if msg.IMAPFolderName != "INBOX" {
			t.Errorf("Expected folder INBOX, got %s", msg.IMAPFolderName)
		}
		if !msg.IsRead {
			t.Error("Expected message to be marked as read")
		}
		if !msg.IsStarred {
			t.Error("Expected message to be starred")
		}
		if msg.MessageIDHeader != "<msg-123@example.com>" {
			t.Errorf("Expected MessageIDHeader '<msg-123@example.com>', got %s", msg.MessageIDHeader)
		}
		if !strings.Contains(msg.FromAddress, "Sender") {
			t.Errorf("Expected FromAddress to contain 'Sender', got %s", msg.FromAddress)
		}
		if len(msg.ToAddresses) != 1 {
			t.Errorf("Expected 1 ToAddress, got %d", len(msg.ToAddresses))
		}
		if msg.Subject != "Test Subject" {
			t.Errorf("Expected Subject 'Test Subject', got %s", msg.Subject)
		}
		if msg.SentAt == nil || !msg.SentAt.Equal(now) {
			t.Error("Expected SentAt to match envelope date")
		}
	})

	t.Run("handles nil message", func(t *testing.T) {
		_, err := ParseMessage(nil, "thread-id", "user-id", "INBOX")
		if err == nil {
			t.Error("Expected error for nil message")
		}
	})

	t.Run("handles message without envelope", func(t *testing.T) {
		imapMsg := &imap.Message{
			Uid:   200,
			Flags: []string{},
		}

		msg, err := ParseMessage(imapMsg, "thread-id", "user-id", "INBOX")
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}

		if msg.IMAPUID != 200 {
			t.Errorf("Expected IMAPUID 200, got %d", msg.IMAPUID)
		}
		if msg.IsRead {
			t.Error("Expected message to not be marked as read")
		}
	})

	t.Run("handles message without Message-ID", func(t *testing.T) {
		imapMsg := &imap.Message{
			Uid:   300,
			Flags: []string{},
			Envelope: &imap.Envelope{
				// No MessageId
				Subject: "Test Subject",
			},
		}

		msg, err := ParseMessage(imapMsg, "thread-id", "user-id", "INBOX")
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}

		if msg.MessageIDHeader != "" {
			t.Errorf("Expected empty MessageIDHeader, got %s", msg.MessageIDHeader)
		}
		if msg.Subject != "Test Subject" {
			t.Errorf("Expected Subject 'Test Subject', got %s", msg.Subject)
		}
	})

	t.Run("handles message with empty body", func(t *testing.T) {
		imapMsg := &imap.Message{
			Uid:   400,
			Flags: []string{},
			Envelope: &imap.Envelope{
				MessageId: "<empty-body@example.com>",
			},
			// No Body or BodyStructure
		}

		msg, err := ParseMessage(imapMsg, "thread-id", "user-id", "INBOX")
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}

		if msg.UnsafeBodyHTML != "" {
			t.Errorf("Expected empty body HTML, got %s", msg.UnsafeBodyHTML)
		}
		if msg.BodyText != "" {
			t.Errorf("Expected empty body text, got %s", msg.BodyText)
		}
	})

	t.Run("handles body parsing errors gracefully", func(t *testing.T) {
		// Create a message with invalid body structure
		imapMsg := &imap.Message{
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
			// Body is nil, which will cause parseBody to fail, but ParseMessage should continue
		}

		msg, err := ParseMessage(imapMsg, "thread-id", "user-id", "INBOX")
		if err != nil {
			t.Fatalf("ParseMessage should not fail on body parsing errors: %v", err)
		}

		// Should still have headers even if body parsing failed
		if msg.Subject != "Test Subject" {
			t.Errorf("Expected Subject 'Test Subject', got %s", msg.Subject)
		}
		if msg.MessageIDHeader != "<invalid-body@example.com>" {
			t.Errorf("Expected MessageIDHeader '<invalid-body@example.com>', got %s", msg.MessageIDHeader)
		}
	})

	t.Run("handles message with attachments", func(t *testing.T) {
		// Note: Testing attachments requires a properly formatted MIME message
		// For now, we test that the function handles messages with BodyStructure
		// that indicates attachments. Full attachment parsing is tested through
		// integration tests with real IMAP messages.
		imapMsg := &imap.Message{
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
		}

		msg, err := ParseMessage(imapMsg, "thread-id", "user-id", "INBOX")
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}

		// Message should be parsed successfully
		if msg.Subject != "Test with Attachments" {
			t.Errorf("Expected Subject 'Test with Attachments', got %s", msg.Subject)
		}
		// Attachments would be parsed from the body if Body is available
		// This is tested through integration tests
	})
}
