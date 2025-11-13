package testutil

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-smtp"
)

// MemoryBackend is a simple in-memory SMTP backend for testing.
type MemoryBackend struct {
	mu       sync.Mutex
	messages []*memoryMessage
}

type memoryMessage struct {
	From string
	To   []string
	Data []byte
}

// NewMemoryBackend creates a new in-memory SMTP backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		messages: make([]*memoryMessage, 0),
	}
}

// NewSession creates a new SMTP session.
func (b *MemoryBackend) NewSession(*smtp.Conn) (smtp.Session, error) {
	return &memorySession{backend: b}, nil
}

// GetMessages returns all received messages.
func (b *MemoryBackend) GetMessages() []*memoryMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.messages
}

// ClearMessages clears all stored messages.
func (b *MemoryBackend) ClearMessages() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = make([]*memoryMessage, 0)
}

type memorySession struct {
	backend *MemoryBackend
	from    string
	to      []string
}

func (s *memorySession) AuthMechanism() (string, bool) {
	return "PLAIN", true
}

func (s *memorySession) Auth(username, password string) error {
	// Accept any credentials for testing
	return nil
}

func (s *memorySession) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *memorySession) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *memorySession) Data(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	s.backend.mu.Lock()
	defer s.backend.mu.Unlock()

	s.backend.messages = append(s.backend.messages, &memoryMessage{
		From: s.from,
		To:   s.to,
		Data: data,
	})

	return nil
}

func (s *memorySession) Reset() {
	s.from = ""
	s.to = nil
}

func (s *memorySession) Logout() error {
	return nil
}

// TestSMTPServer represents a test SMTP server instance.
type TestSMTPServer struct {
	Server   *smtp.Server
	Address  string
	Backend  *MemoryBackend
	cleanup  func()
	username string
	password string
}

// NewTestSMTPServer creates a new test SMTP server with an in-memory backend.
// Returns the server instance.
// The memory backend accepts any username/password combination for testing.
func NewTestSMTPServer(t *testing.T) *TestSMTPServer {
	t.Helper()

	// Create an in-memory backend
	be := NewMemoryBackend()

	// Create server
	s := smtp.NewServer(be)
	s.Addr = ":0" // Random port
	s.AllowInsecureAuth = true
	s.Domain = "localhost"

	// Start server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	addr := listener.Addr().String()

	// Start server in goroutine
	go func() {
		if err := s.Serve(listener); err != nil {
			t.Logf("SMTP server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
		err := s.Close()
		if err != nil {
			t.Logf("Failed to close SMTP server: %v", err)
			return
		}
	}

	// Memory backend accepts any credentials for testing
	username := "test-user"
	password := "test-pass"

	return &TestSMTPServer{
		Server:   s,
		Address:  addr,
		Backend:  be,
		cleanup:  cleanup,
		username: username,
		password: password,
	}
}

// Close shuts down the test SMTP server.
func (s *TestSMTPServer) Close() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

// Username returns the test username.
func (s *TestSMTPServer) Username() string {
	return s.username
}

// Password returns the test password.
func (s *TestSMTPServer) Password() string {
	return s.password
}

// GetMessages returns all messages received by the server.
func (s *TestSMTPServer) GetMessages() []*memoryMessage {
	return s.Backend.GetMessages()
}

// ClearMessages clears all stored messages.
func (s *TestSMTPServer) ClearMessages() {
	s.Backend.ClearMessages()
}

// NewTestSMTPServerForE2E creates a new test SMTP server for E2E tests (non-test context).
// Returns the server instance. The memory backend accepts any username/password combination.
// Uses a fixed port (1025) for E2E tests so Playwright can connect to it.
func NewTestSMTPServerForE2E() (*TestSMTPServer, error) {
	// Create an in-memory backend
	be := NewMemoryBackend()

	// Create server
	s := smtp.NewServer(be)
	s.Addr = ":1025" // Fixed port for E2E tests
	s.AllowInsecureAuth = true
	s.Domain = "localhost"

	// Start server on fixed port for E2E tests
	listener, err := net.Listen("tcp", "127.0.0.1:1025")
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

	// Memory backend accepts any credentials for testing
	username := "test-user"
	password := "test-pass"

	return &TestSMTPServer{
		Server:   s,
		Address:  addr,
		Backend:  be,
		cleanup:  cleanup,
		username: username,
		password: password,
	}, nil
}
