package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vdavid/vmail/backend/internal/config"
	"github.com/vdavid/vmail/backend/internal/db"
)

func getTestConfig() *config.Config {
	pgHost := os.Getenv("PGHOST")
	pgPort := os.Getenv("PGPORT")
	pgUser := os.Getenv("PGUSER")
	pgPassword := os.Getenv("PGPASSWORD")
	pgDatabase := os.Getenv("PGDATABASE")

	if pgHost == "" {
		pgHost = "localhost"
	}
	if pgPort == "" {
		pgPort = "5432"
	}
	if pgUser == "" {
		pgUser = "postgres"
	}
	if pgPassword == "" {
		pgPassword = "postgres"
	}
	if pgDatabase == "" {
		pgDatabase = "postgres"
	}

	sslMode := "disable"
	if pgHost != "localhost" && pgHost != "127.0.0.1" {
		sslMode = "require"
	}

	return &config.Config{
		Environment:   "test",
		EncryptionKey: "test-encryption-key",
		AutheliaURL:   "http://authelia:9091",
		DBHost:        pgHost,
		DBPort:        pgPort,
		DBUsername:    pgUser,
		DBPassword:    pgPassword,
		DBName:        pgDatabase,
		DBSSLMode:     sslMode,
		Port:          "8080",
		Timezone:      "UTC",
	}
}

func TestHandleRoot(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handleRoot(w, req)

	res := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatalf("failed to close response body: %v", err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	contentType := res.Header.Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("expected Content-Type 'text/plain', got '%s'", contentType)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	expected := "V-Mail API is running"
	if string(body) != expected {
		t.Errorf("expected body '%s', got '%s'", expected, string(body))
	}
}

func TestNewServer(t *testing.T) {
	cfg := getTestConfig()
	ctx := context.Background()

	pool, err := db.NewConnection(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.CloseConnection(pool)

	server := NewServer(cfg, pool)

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.ServeHTTP(w, req)

	res := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatalf("failed to close response body: %v", err)
		}
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	expected := "V-Mail API is running"
	if string(body) != expected {
		t.Errorf("expected body '%s', got '%s'", expected, string(body))
	}
}

func TestMainWithConfig(t *testing.T) {
	_ = os.Setenv("VMAIL_ENV", "production")
	_ = os.Setenv("VMAIL_ENCRYPTION_KEY", "test-key-123456789012345678901234")
	_ = os.Setenv("AUTHELIA_URL", "http://authelia:9091")
	_ = os.Setenv("VMAIL_DB_HOST", os.Getenv("PGHOST"))
	_ = os.Setenv("VMAIL_DB_PORT", os.Getenv("PGPORT"))
	_ = os.Setenv("VMAIL_DB_USER", os.Getenv("PGUSER"))
	_ = os.Setenv("VMAIL_DB_PASSWORD", os.Getenv("PGPASSWORD"))
	_ = os.Setenv("VMAIL_DB_NAME", os.Getenv("PGDATABASE"))

	sslMode := "disable"
	if os.Getenv("PGHOST") != "localhost" && os.Getenv("PGHOST") != "127.0.0.1" && os.Getenv("PGHOST") != "" {
		sslMode = "require"
	}
	_ = os.Setenv("VMAIL_DB_SSLMODE", sslMode)
	_ = os.Setenv("PORT", "9999")

	defer func() {
		_ = os.Unsetenv("VMAIL_ENV")
		_ = os.Unsetenv("VMAIL_ENCRYPTION_KEY")
		_ = os.Unsetenv("AUTHELIA_URL")
		_ = os.Unsetenv("VMAIL_DB_HOST")
		_ = os.Unsetenv("VMAIL_DB_PORT")
		_ = os.Unsetenv("VMAIL_DB_USER")
		_ = os.Unsetenv("VMAIL_DB_PASSWORD")
		_ = os.Unsetenv("VMAIL_DB_NAME")
		_ = os.Unsetenv("VMAIL_DB_SSLMODE")
		_ = os.Unsetenv("PORT")
	}()

	cfg, err := config.NewConfig()
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	if cfg.Port != "9999" {
		t.Errorf("expected port '9999', got '%s'", cfg.Port)
	}

	ctx := context.Background()
	pool, err := db.NewConnection(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create database connection: %v", err)
	}
	defer db.CloseConnection(pool)

	server := NewServer(cfg, pool)
	if server == nil {
		t.Fatal("NewServer() returned nil with valid config")
	}
}
