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
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/vdavid/vmail/backend/internal/api"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/config"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
	"github.com/vdavid/vmail/backend/internal/models"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func main() {
	ctx := context.Background()

	// Setup environment variables
	if err := setupTestEnvironment(); err != nil {
		log.Fatalf("Failed to setup test environment: %v", err)
	}

	// Start Postgres database
	postgresContainer, connStr, err := startPostgres(ctx)
	if err != nil {
		log.Fatalf("Failed to start Postgres: %v", err)
	}
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate Postgres container: %v", err)
		}
	}()

	// Start test mail servers
	imapServer, smtpServer, err := startMailServers()
	if err != nil {
		log.Fatalf("Failed to start mail servers: %v", err)
	}
	defer imapServer.Close()
	defer smtpServer.Close()

	// Seed test data into IMAP server
	if err := seedTestData(imapServer); err != nil {
		log.Fatalf("Failed to seed test data: %v", err)
	}

	// Setup database connection and run migrations
	cfg, pool, err := setupDatabase(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}
	defer pool.Close()

	// Seed user settings and sync messages
	if err := setupTestUser(ctx, pool, cfg, imapServer, smtpServer); err != nil {
		log.Fatalf("Failed to setup test user: %v", err)
	}

	// Start HTTP server
	if err := startHTTPServer(cfg, pool, imapServer, smtpServer); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// setupTestEnvironment sets up required environment variables for the test server.
func setupTestEnvironment() error {
	if err := os.Setenv("VMAIL_TEST_MODE", "true"); err != nil {
		return fmt.Errorf("failed to set VMAIL_TEST_MODE: %w", err)
	}

	testEncryptionKey := "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM="
	if err := os.Setenv("VMAIL_ENCRYPTION_KEY_BASE64", testEncryptionKey); err != nil {
		return fmt.Errorf("failed to set VMAIL_ENCRYPTION_KEY_BASE64: %w", err)
	}

	if err := os.Setenv("AUTHELIA_URL", "http://localhost:9091"); err != nil {
		return fmt.Errorf("failed to set AUTHELIA_URL: %w", err)
	}

	if err := os.Setenv("VMAIL_DB_PASSWORD", "vmail"); err != nil {
		return fmt.Errorf("failed to set VMAIL_DB_PASSWORD: %w", err)
	}

	return nil
}

// startPostgres starts a test Postgres database using testcontainers.
func startPostgres(ctx context.Context) (testcontainers.Container, string, error) {
	log.Println("Starting test Postgres database...")
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("vmail_test"),
		postgres.WithUsername("vmail"),
		postgres.WithPassword("vmail"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to start Postgres container: %w", err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", fmt.Errorf("failed to get connection string: %w", err)
	}

	log.Println("Test Postgres database started")
	return postgresContainer, connStr, nil
}

// startMailServers starts test IMAP and SMTP servers.
func startMailServers() (*testutil.TestIMAPServer, *testutil.TestSMTPServer, error) {
	log.Println("Starting test IMAP server...")
	imapServer, err := testutil.NewTestIMAPServerForE2E()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start test IMAP server: %w", err)
	}
	log.Printf("Test IMAP server started on %s", imapServer.Address)

	log.Println("Starting test SMTP server...")
	smtpServer, err := testutil.NewTestSMTPServerForE2E()
	if err != nil {
		imapServer.Close()
		return nil, nil, fmt.Errorf("failed to start test SMTP server: %w", err)
	}
	log.Printf("Test SMTP server started on %s", smtpServer.Address)

	return imapServer, smtpServer, nil
}

// setupDatabase creates a database connection pool and runs migrations.
func setupDatabase(ctx context.Context, connStr string) (*config.Config, *pgxpool.Pool, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := testutil.RunMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Successfully connected to database and ran migrations")
	return cfg, pool, nil
}

// setupTestUser seeds user settings and syncs IMAP messages for the test user.
func setupTestUser(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, imapServer *testutil.TestIMAPServer, smtpServer *testutil.TestSMTPServer) error {
	if err := seedUserSettings(ctx, pool, cfg, imapServer, smtpServer); err != nil {
		return fmt.Errorf("failed to seed user settings: %w", err)
	}
	log.Println("User settings seeded for test user")

	testEmail := "test@example.com"
	userID, err := db.GetOrCreateUser(ctx, pool, testEmail)
	if err != nil {
		return fmt.Errorf("failed to get test user: %w", err)
	}

	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKeyBase64)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	imapPool := imap.NewPool()
	defer imapPool.Close()

	imapService := imap.NewService(pool, imapPool, encryptor)
	if err := imapService.SyncThreadsForFolder(ctx, userID, "INBOX"); err != nil {
		log.Printf("Warning: Failed to sync INBOX folder: %v", err)
	} else {
		log.Println("Synced INBOX folder to database")
	}

	return nil
}

// startHTTPServer starts the HTTP server and waits for shutdown signals.
func startHTTPServer(cfg *config.Config, dbPool *pgxpool.Pool, imapServer *testutil.TestIMAPServer, smtpServer *testutil.TestSMTPServer) error {
	server := NewServer(cfg, dbPool)
	address := ":" + cfg.Port

	log.Printf("V-Mail test server starting on %s", address)
	log.Printf("Test IMAP server: %s (username: %s, password: %s)", imapServer.Address, imapServer.Username(), imapServer.Password())
	log.Printf("Test SMTP server: %s (username: %s, password: %s)", smtpServer.Address, smtpServer.Username(), smtpServer.Password())
	log.Println("Server ready for E2E tests. Press Ctrl+C to stop.")

	serverErr := make(chan error, 1)
	go func() {
		if err := http.ListenAndServe(address, server); err != nil {
			serverErr <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
		return nil
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}
}

// NewServer creates and returns a new HTTP handler for the V-Mail API server.
func NewServer(cfg *config.Config, dbPool *pgxpool.Pool) http.Handler {
	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKeyBase64)
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}

	imapPool := imap.NewPoolWithMaxWorkers(cfg.IMAPMaxWorkers)
	imapService := imap.NewService(dbPool, imapPool, encryptor)

	authHandler := api.NewAuthHandler(dbPool)
	settingsHandler := api.NewSettingsHandler(dbPool, encryptor)
	foldersHandler := api.NewFoldersHandler(dbPool, encryptor, imapPool)
	threadsHandler := api.NewThreadsHandler(dbPool, encryptor, imapService)
	threadHandler := api.NewThreadHandler(dbPool, encryptor, imapService)
	searchHandler := api.NewSearchHandler(dbPool, encryptor, imapService)

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
	// Ensure INBOX exists (INBOX is identified by name, not SPECIAL-USE)
	if err := imapServer.EnsureINBOXForE2E(); err != nil {
		return fmt.Errorf("failed to ensure INBOX: %w", err)
	}

	// Create folders with SPECIAL-USE attributes for testing
	// Note: The go-imap memory backend should support SPECIAL-USE if the server is configured correctly
	// We create the folders here; the SPECIAL-USE attributes should be set by the backend
	folders := []struct {
		name string
		attr string
	}{
		{"Sent", "\\Sent"},
		{"Drafts", "\\Drafts"},
		{"Trash", "\\Trash"},
		{"Spam", "\\Junk"},
		{"Archive", "\\Archive"},
	}

	client, err := imapServer.ConnectForE2E()
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer func() {
		_ = client.Logout()
	}()

	for _, folder := range folders {
		// Create folder if it doesn't exist
		err = client.Create(folder.name)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			log.Printf("Warning: Failed to create folder %s: %v", folder.name, err)
		}
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

// seedUserSettings creates user settings for the test user so "existing user" tests work
func seedUserSettings(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, imapServer *testutil.TestIMAPServer, smtpServer *testutil.TestSMTPServer) error {
	// Get or create test user
	testEmail := "test@example.com"
	userID, err := db.GetOrCreateUser(ctx, pool, testEmail)
	if err != nil {
		return fmt.Errorf("failed to get or create user: %w", err)
	}

	// Create encryptor to encrypt passwords
	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKeyBase64)
	if err != nil {
		return fmt.Errorf("failed to create encryptor: %w", err)
	}

	// Encrypt passwords
	encryptedIMAPPassword, err := encryptor.Encrypt(imapServer.Password())
	if err != nil {
		return fmt.Errorf("failed to encrypt IMAP password: %w", err)
	}

	encryptedSMTPPassword, err := encryptor.Encrypt(smtpServer.Password())
	if err != nil {
		return fmt.Errorf("failed to encrypt SMTP password: %w", err)
	}

	// Create user settings
	settings := &models.UserSettings{
		UserID:                   userID,
		IMAPServerHostname:       imapServer.Address,
		IMAPUsername:             imapServer.Username(),
		EncryptedIMAPPassword:    encryptedIMAPPassword,
		SMTPServerHostname:       smtpServer.Address,
		SMTPUsername:             smtpServer.Username(),
		EncryptedSMTPPassword:    encryptedSMTPPassword,
		UndoSendDelaySeconds:     20,
		PaginationThreadsPerPage: 100,
	}

	// Save user settings
	if err := db.SaveUserSettings(ctx, pool, settings); err != nil {
		return fmt.Errorf("failed to save user settings: %w", err)
	}

	return nil
}
