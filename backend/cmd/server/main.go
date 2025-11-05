package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/api"
	"github.com/vdavid/vmail/backend/internal/auth"
	"github.com/vdavid/vmail/backend/internal/config"
	"github.com/vdavid/vmail/backend/internal/crypto"
	"github.com/vdavid/vmail/backend/internal/db"
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

	addr := ":" + cfg.Port
	log.Printf("V-Mail backend server starting on %s (environment: %s)", addr, cfg.Environment)

	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func NewServer(cfg *config.Config, pool *pgxpool.Pool) http.Handler {
	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("Failed to create encryptor: %v", err)
	}

	authHandler := api.NewAuthHandler(pool)
	settingsHandler := api.NewSettingsHandler(pool, encryptor)

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

	return mux
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "V-Mail API is running")
}
