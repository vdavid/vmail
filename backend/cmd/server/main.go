package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdavid/vmail/backend/internal/config"
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
	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)

	return mux
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "V-Mail API is running")
}
