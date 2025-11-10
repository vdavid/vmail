package testutil

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/memory"
	imapclient "github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/server"
)

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
