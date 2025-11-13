package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vdavid/vmail/backend/internal/config"
	"github.com/vdavid/vmail/backend/internal/testutil"
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
		Environment:         "test",
		EncryptionKeyBase64: "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
		AutheliaURL:         "http://authelia:9091",
		DBHost:              pgHost,
		DBPort:              pgPort,
		DBUsername:          pgUser,
		DBPassword:          pgPassword,
		DBName:              pgDatabase,
		DBSSLMode:           sslMode,
		Port:                "11764",
		Timezone:            "UTC",
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
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	cfg := getTestConfig()
	ctx := context.Background()

	// Use the test pool directly instead of creating a new connection
	err := pool.Ping(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

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
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	cfg := getTestConfig()
	ctx := context.Background()

	err := pool.Ping(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	server := NewServer(cfg, pool)
	if server == nil {
		t.Fatal("NewServer() returned nil with valid config")
	}
}
