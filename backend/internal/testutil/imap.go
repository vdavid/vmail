package testutil

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/backend/memory"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/server"
)

// Ensure specialUseExtension implements server.Extension interface
var _ server.Extension = (*specialUseExtension)(nil)

// specialUseExtension is a simple extension that advertises SPECIAL-USE capability.
type specialUseExtension struct{}

// Capabilities returns the SPECIAL-USE capability.
func (e *specialUseExtension) Capabilities(server.Conn) []string {
	return []string{"SPECIAL-USE"}
}

// Command returns nil (no custom commands needed for SPECIAL-USE).
func (e *specialUseExtension) Command(string) server.HandlerFactory {
	return nil
}

// specialUseBackend wraps a memory backend and adds SPECIAL-USE support.
type specialUseBackend struct {
	backend.Backend
	memoryBackend *memory.Backend
}

// Ensure specialUseBackend implements backend.Backend interface
var _ backend.Backend = (*specialUseBackend)(nil)

// Login wraps the memory backend's Login and returns a user with SPECIAL-USE support.
func (b *specialUseBackend) Login(connInfo *imap.ConnInfo, username, password string) (backend.User, error) {
	user, err := b.memoryBackend.Login(connInfo, username, password)
	if err != nil {
		return nil, err
	}
	return &specialUseUser{User: user}, nil
}

// specialUseUser wraps a memory user and adds SPECIAL-USE mailbox attributes.
type specialUseUser struct {
	backend.User
}

// GetMailbox wraps the memory user's GetMailbox and adds SPECIAL-USE attributes.
func (u *specialUseUser) GetMailbox(name string) (backend.Mailbox, error) {
	mb, err := u.User.GetMailbox(name)
	if err != nil {
		return nil, err
	}

	// Add SPECIAL-USE attributes based on mailbox name
	attrs := []string{}
	switch name {
	case "Sent":
		attrs = append(attrs, "\\Sent")
	case "Drafts":
		attrs = append(attrs, "\\Drafts")
	case "Trash":
		attrs = append(attrs, "\\Trash")
	case "Spam":
		attrs = append(attrs, "\\Junk")
	case "Archive":
		attrs = append(attrs, "\\Archive")
	}

	return &specialUseMailbox{Mailbox: mb, attrs: attrs}, nil
}

// ListMailboxes wraps the memory user's ListMailboxes and returns mailboxes with SPECIAL-USE support.
func (u *specialUseUser) ListMailboxes(subscribed bool) ([]backend.Mailbox, error) {
	mailboxes, err := u.User.ListMailboxes(subscribed)
	if err != nil {
		return nil, err
	}

	// Wrap each mailbox with SPECIAL-USE support
	result := make([]backend.Mailbox, len(mailboxes))
	for i, mb := range mailboxes {
		info, err := mb.Info()
		if err != nil {
			return nil, err
		}

		// Add SPECIAL-USE attributes based on mailbox name
		attrs := []string{}
		switch info.Name {
		case "Sent":
			attrs = append(attrs, "\\Sent")
		case "Drafts":
			attrs = append(attrs, "\\Drafts")
		case "Trash":
			attrs = append(attrs, "\\Trash")
		case "Spam":
			attrs = append(attrs, "\\Junk")
		case "Archive":
			attrs = append(attrs, "\\Archive")
		}

		result[i] = &specialUseMailbox{Mailbox: mb, attrs: attrs}
	}

	return result, nil
}

// specialUseMailbox wraps a memory mailbox and adds SPECIAL-USE attributes.
type specialUseMailbox struct {
	backend.Mailbox
	attrs []string
}

// Info returns mailbox info with SPECIAL-USE attributes.
func (m *specialUseMailbox) Info() (*imap.MailboxInfo, error) {
	info, err := m.Mailbox.Info()
	if err != nil {
		return nil, err
	}
	info.Attributes = append(info.Attributes, m.attrs...)
	return info, nil
}

// TestIMAPServer represents a test IMAP server instance.
type TestIMAPServer struct {
	Server   *server.Server
	Address  string
	Backend  *memory.Backend
	cleanup  func()
	username string
	password string
}

// NewTestIMAPServer creates a new test IMAP server with an in-memory backend.
// Returns the server instance and cleanup function.
// The memory backend creates a default user with username "username" and password "password".
//
// Note: This function is intended for use in test files (requires *testing.T).
// For E2E tests that don't have a testing context, use NewTestIMAPServerForE2E instead.
func NewTestIMAPServer(t *testing.T) *TestIMAPServer {
	t.Helper()

	// Create an in-memory backend
	be := memory.New()

	// Create server
	s := server.New(be)
	s.AllowInsecureAuth = true

	// Start server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	addr := listener.Addr().String()

	// Start server in goroutine
	go func() {
		if err := s.Serve(listener); err != nil {
			t.Logf("IMAP server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
		err := s.Close()
		if err != nil {
			t.Logf("Failed to close IMAP server: %v", err)
			return
		}
	}

	// Memory backend creates a default user with these credentials
	username := "username"
	password := "password"

	return &TestIMAPServer{
		Server:   s,
		Address:  addr,
		Backend:  be,
		cleanup:  cleanup,
		username: username,
		password: password,
	}
}

// Close shuts down the test IMAP server.
func (s *TestIMAPServer) Close() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

// Username returns the default test username.
func (s *TestIMAPServer) Username() string {
	return s.username
}

// Password returns the default test password.
func (s *TestIMAPServer) Password() string {
	return s.password
}

// Connect creates a new IMAP client connection to the test server.
func (s *TestIMAPServer) Connect(t *testing.T) (*imapclient.Client, func()) {
	t.Helper()

	client, err := imapclient.Dial(s.Address)
	if err != nil {
		t.Fatalf("Failed to connect to test server: %v", err)
	}

	if err := client.Login(s.username, s.password); err != nil {
		_ = client.Logout()
		t.Fatalf("Failed to login: %v", err)
	}

	cleanup := func() {
		_ = client.Logout()
	}

	return client, cleanup
}

// EnsureINBOX ensures the INBOX folder exists for the default user.
func (s *TestIMAPServer) EnsureINBOX(t *testing.T) {
	t.Helper()

	client, cleanup := s.Connect(t)
	defer cleanup()

	_, err := client.Select("INBOX", false)
	if err != nil {
		// Create INBOX if it doesn't exist
		err = client.Create("INBOX")
		if err != nil {
			t.Fatalf("Failed to create INBOX: %v", err)
		}
		_, err = client.Select("INBOX", false)
		if err != nil {
			t.Fatalf("Failed to select INBOX: %v", err)
		}
	}
}

// AddMessage adds a test message to the specified folder and returns its UID.
func (s *TestIMAPServer) AddMessage(t *testing.T, folderName, messageID, subject, from, to string, sentAt time.Time) uint32 {
	t.Helper()

	client, cleanup := s.Connect(t)
	defer cleanup()

	// Select the folder
	_, err := client.Select(folderName, false)
	if err != nil {
		t.Fatalf("Failed to select folder: %v", err)
	}

	// Create a simple RFC 822 message
	messageBody := fmt.Sprintf(`Message-ID: %s
Date: %s
From: %s
To: %s
Subject: %s
Content-Type: text/plain; charset=utf-8

Test message body.
`, messageID, sentAt.Format(time.RFC1123Z), from, to, subject)

	// Append the message to the folder
	flags := []string{imap.SeenFlag}
	now := time.Now()
	err = client.Append(folderName, flags, now, strings.NewReader(messageBody))
	if err != nil {
		t.Fatalf("Failed to append message: %v", err)
	}

	// Search for the message we just added to get its UID
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("Message-ID", messageID)
	uids, err := client.UidSearch(criteria)
	if err != nil {
		t.Fatalf("Failed to search for message: %v", err)
	}

	if len(uids) == 0 {
		t.Fatalf("Message not found after append")
	}

	return uids[0]
}

// NewTestIMAPServerForE2E creates a new test IMAP server for E2E tests (non-test context).
// Returns the server instance. The memory backend creates a default user with
// username "username" and password "password".
// Uses a fixed port (1143) for E2E tests so Playwright can connect to it.
// The server includes SPECIAL-USE support for folder role detection.
func NewTestIMAPServerForE2E() (*TestIMAPServer, error) {
	// Create an in-memory backend
	memoryBackend := memory.New()

	// Wrap it with SPECIAL-USE support
	be := &specialUseBackend{
		Backend:       memoryBackend,
		memoryBackend: memoryBackend,
	}

	// Create server
	s := server.New(be)
	s.AllowInsecureAuth = true

	// Enable SPECIAL-USE extension to advertise the capability
	s.Enable(&specialUseExtension{})

	// Start server on fixed port for E2E tests
	listener, err := net.Listen("tcp", "127.0.0.1:1143")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	addr := listener.Addr().String()

	// Start server in goroutine
	go func() {
		if err := s.Serve(listener); err != nil {
			// Server closed, ignore error
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
		_ = s.Close()
	}

	// Memory backend creates a default user with these credentials
	username := "username"
	password := "password"

	return &TestIMAPServer{
		Server:   s,
		Address:  addr,
		Backend:  memoryBackend, // Store the original memory backend for direct access if needed
		cleanup:  cleanup,
		username: username,
		password: password,
	}, nil
}

// ConnectForE2E creates a new IMAP client connection to the test server (non-test context).
func (s *TestIMAPServer) ConnectForE2E() (*imapclient.Client, error) {
	client, err := imapclient.Dial(s.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test server: %w", err)
	}

	if err := client.Login(s.username, s.password); err != nil {
		_ = client.Logout()
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return client, nil
}

// EnsureINBOXForE2E ensures the INBOX folder exists for the default user (non-test context).
func (s *TestIMAPServer) EnsureINBOXForE2E() error {
	client, err := s.ConnectForE2E()
	if err != nil {
		return err
	}
	defer func() {
		_ = client.Logout()
	}()

	_, err = client.Select("INBOX", false)
	if err != nil {
		// Create INBOX if it doesn't exist
		err = client.Create("INBOX")
		if err != nil {
			return fmt.Errorf("failed to create INBOX: %w", err)
		}
		_, err = client.Select("INBOX", false)
		if err != nil {
			return fmt.Errorf("failed to select INBOX: %w", err)
		}
	}

	return nil
}

// AddMessageForE2E adds a test message to the specified folder and returns its UID (non-test context).
func (s *TestIMAPServer) AddMessageForE2E(folderName, messageID, subject, from, to string, sentAt time.Time) (uint32, error) {
	client, err := s.ConnectForE2E()
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = client.Logout()
	}()

	// Select the folder
	_, err = client.Select(folderName, false)
	if err != nil {
		return 0, fmt.Errorf("failed to select folder: %w", err)
	}

	// Create a simple RFC 822 message
	messageBody := fmt.Sprintf(`Message-ID: %s
Date: %s
From: %s
To: %s
Subject: %s
Content-Type: text/plain; charset=utf-8

Test message body.
`, messageID, sentAt.Format(time.RFC1123Z), from, to, subject)

	// Append the message to the folder
	flags := []string{imap.SeenFlag}
	now := time.Now()
	err = client.Append(folderName, flags, now, strings.NewReader(messageBody))
	if err != nil {
		return 0, fmt.Errorf("failed to append message: %w", err)
	}

	// Search for the message we just added to get its UID
	criteria := imap.NewSearchCriteria()
	criteria.Header.Add("Message-ID", messageID)
	uids, err := client.UidSearch(criteria)
	if err != nil {
		return 0, fmt.Errorf("failed to search for message: %w", err)
	}

	if len(uids) == 0 {
		return 0, fmt.Errorf("message not found after append")
	}

	return uids[0], nil
}

// CreateFolderWithSpecialUse creates a folder with SPECIAL-USE attributes (non-test context).
func (s *TestIMAPServer) CreateFolderWithSpecialUse(folderName string) error {
	client, err := s.ConnectForE2E()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		_ = client.Logout()
	}()

	// Create the folder
	err = client.Create(folderName)
	if err != nil {
		return fmt.Errorf("failed to create folder %s: %w", folderName, err)
	}

	// Set SPECIAL-USE attribute using SETMETADATA (if supported) or LIST-EXTENDED
	// Note: go-imap memory backend may not support SETMETADATA, but it should
	// support SPECIAL-USE attributes when listing if we configure them properly.
	// For the memory backend, we'll rely on the server's default behavior.
	// The memory backend should support SPECIAL-USE if the server is configured correctly.

	return nil
}
