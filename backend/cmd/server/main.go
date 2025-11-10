package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/api"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/config"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/imap"
)

func main() {
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

	log.Printf("Successfully connected to database")

	server := NewServer(cfg, pool)

	address := ":" + cfg.Port
	log.Printf("V-Mail backend server starting on %s (environment: %s)", address, cfg.Environment)

	if err := http.ListenAndServe(address, server); err != nil {
		log.Fatalf("Server failed to start: %v", err)
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
	_, _ = fmt.Fprintf(w, "V-Mail API is running")
}
