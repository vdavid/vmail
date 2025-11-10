package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/api"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/config"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func main() {
	// Set test mode environment variable
	os.Setenv("VMAIL_TEST_MODE", "true")

	// Start test IMAP server
	log.Println("Starting test IMAP server...")
	imapServer, err := testutil.NewTestIMAPServerForE2E()
	if err != nil {
		log.Fatalf("Failed to start test IMAP server: %v", err)
	}
	defer imapServer.Close()
	log.Printf("Test IMAP server started on %s", imapServer.Address)

	// Start test SMTP server
	log.Println("Starting test SMTP server...")
	smtpServer, err := testutil.NewTestSMTPServerForE2E()
	if err != nil {
		log.Fatalf("Failed to start test SMTP server: %v", err)
	}
	defer smtpServer.Close()
	log.Printf("Test SMTP server started on %s", smtpServer.Address)

	// Ensure INBOX exists and seed test data
	log.Println("Seeding test data...")
	if err := seedTestData(imapServer); err != nil {
		log.Fatalf("Failed to seed test data: %v", err)
	}
	log.Println("Test data seeded successfully")

	// Set Authelia URL to a mock (tests don't need real Authelia)
	// Must be set before NewConfig() because validation requires it
	os.Setenv("AUTHELIA_URL", "http://localhost:9091")

	// Load config (will use .env file if in development mode)
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewConnection(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.CloseConnection(pool)

	log.Println("Successfully connected to database")

	// Create and start backend server
	server := NewServer(cfg, pool)

	address := ":" + cfg.Port
	log.Printf("V-Mail test server starting on %s", address)
	log.Printf("Test IMAP server: %s (username: %s, password: %s)", imapServer.Address, imapServer.Username(), imapServer.Password())
	log.Printf("Test SMTP server: %s (username: %s, password: %s)", smtpServer.Address, smtpServer.Username(), smtpServer.Password())
	log.Println("Server ready for E2E tests. Press Ctrl+C to stop.")

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := http.ListenAndServe(address, server); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	}
}

// NewServer creates and returns a new HTTP handler for the V-Mail API server.
func NewServer(cfg *config.Config, pool *pgxpool.Pool) http.Handler {
	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKeyBase64)
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}

	imapPool := imap.NewPool()
	imapService := imap.NewService(pool, encryptor)

	authHandler := api.NewAuthHandler(pool)
	settingsHandler := api.NewSettingsHandler(pool, encryptor)
	foldersHandler := api.NewFoldersHandler(pool, encryptor, imapPool)
	threadsHandler := api.NewThreadsHandler(pool, encryptor, imapService)
	threadHandler := api.NewThreadHandler(pool, encryptor, imapService)
	searchHandler := api.NewSearchHandler(pool, encryptor, imapService)

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)

	mux.Handle("/api/v1/auth/status", auth.RequireAuth(http.HandlerFunc(authHandler.GetAuthStatus)))
	mux.Handle("/api/v1/settings", auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			settingsHandler.GetSettings(w, r)
		case http.MethodPost:
			settingsHandler.PostSettings(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/v1/folders", auth.RequireAuth(http.HandlerFunc(foldersHandler.GetFolders)))
	mux.Handle("/api/v1/threads", auth.RequireAuth(http.HandlerFunc(threadsHandler.GetThreads)))
	mux.Handle("/api/v1/search", auth.RequireAuth(http.HandlerFunc(searchHandler.Search)))

	// Handle /api/v1/thread/{thread_id} pattern
	mux.Handle("/api/v1/thread/", auth.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract thread_id from the path
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/thread/")
		if path == "" || path == r.URL.Path {
			http.Error(w, "thread_id is required", http.StatusBadRequest)
			return
		}
		// Set the thread_id in the URL path for the handler to use
		r.URL.Path = "/api/v1/thread/" + path
		threadHandler.GetThread(w, r)
	})))

	return mux
}

func handleRoot(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "V-Mail Test Server is running")
}

// seedTestData seeds the IMAP server with test messages matching e2e/fixtures/test-data.ts
func seedTestData(imapServer *testutil.TestIMAPServer) error {
	// Ensure INBOX exists
	if err := imapServer.EnsureINBOXForE2E(); err != nil {
		return fmt.Errorf("failed to ensure INBOX: %w", err)
	}

	// Seed test messages matching the TypeScript test data
	messages := []struct {
		messageID string
		subject   string
		from      string
		to        string
		body      string
		sentAt    time.Time
	}{
		{
			messageID: "<msg1@test>",
			subject:   "Welcome to V-Mail",
			from:      "sender@example.com",
			to:        "test@example.com",
			body:      "This is a test message.",
			sentAt:    time.Now().Add(-2 * time.Hour),
		},
		{
			messageID: "<msg2@test>",
			subject:   "Meeting Tomorrow",
			from:      "colleague@example.com",
			to:        "test@example.com",
			body:      "Don't forget about the meeting tomorrow at 2 PM.",
			sentAt:    time.Now().Add(-1 * time.Hour),
		},
		{
			messageID: "<msg3@test>",
			subject:   "Special Report Q3",
			from:      "reports@example.com",
			to:        "test@example.com",
			body:      "Here is the Q3 report you requested.",
			sentAt:    time.Now(),
		},
	}

	for _, msg := range messages {
		_, err := imapServer.AddMessageForE2E("INBOX", msg.messageID, msg.subject, msg.from, msg.to, msg.sentAt)
		if err != nil {
			return fmt.Errorf("failed to add message %s: %w", msg.messageID, err)
		}
	}

	return nil
}
