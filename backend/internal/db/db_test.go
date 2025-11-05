package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/config"
)

func getTestDBConfig() *config.Config {
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
		EncryptionKey: "test-key",
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

func TestNewConnection(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test: DATABASE_URL not set")
	}

	cfg := getTestDBConfig()
	ctx := context.Background()

	pool, err := NewConnection(ctx, cfg)
	if err != nil {
		t.Fatalf("NewConnection() failed: %v", err)
	}
	defer CloseConnection(pool)

	if pool == nil {
		t.Fatal("NewConnection() returned nil pool")
	}

	err = pool.Ping(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}

func TestNewConnectionWithTimeout(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test: DATABASE_URL not set")
	}

	cfg := getTestDBConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := NewConnection(ctx, cfg)
	if err != nil {
		t.Fatalf("NewConnection() failed: %v", err)
	}
	defer CloseConnection(pool)

	if pool == nil {
		t.Fatal("NewConnection() returned nil pool")
	}
}

func TestNewConnectionInvalidConfig(t *testing.T) {
	cfg := &config.Config{
		DBHost:     "invalid-host-that-does-not-exist",
		DBPort:     "5432",
		DBUsername: "invalid",
		DBPassword: "invalid",
		DBName:     "invalid",
		DBSSLMode:  "disable",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewConnection(ctx, cfg)
	if err == nil {
		t.Fatal("Expected NewConnection() to fail with invalid config, but it succeeded")
	}
}

func TestCloseConnection(t *testing.T) {
	CloseConnection(nil)

	if os.Getenv("DATABASE_URL") == "" && os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test: DATABASE_URL not set")
	}

	cfg := getTestDBConfig()
	ctx := context.Background()

	pool, err := NewConnection(ctx, cfg)
	if err != nil {
		t.Fatalf("NewConnection() failed: %v", err)
	}

	CloseConnection(pool)

	err = pool.Ping(ctx)
	if err == nil {
		t.Fatal("Expected Ping() to fail after pool was closed")
	}
}

func TestConnectionPoolProperties(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("SKIP_DB_TESTS") != "" {
		t.Skip("Skipping database test: DATABASE_URL not set")
	}

	cfg := getTestDBConfig()
	ctx := context.Background()

	pool, err := NewConnection(ctx, cfg)
	if err != nil {
		t.Fatalf("NewConnection() failed: %v", err)
	}
	defer CloseConnection(pool)

	stats := pool.Stat()
	if stats.MaxConns() != 25 {
		t.Errorf("Expected MaxConns to be 25, got %d", stats.MaxConns())
	}
}
