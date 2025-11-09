package db

import (
	"context"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/config"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestNewConnection(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	// Test that we can ping the database
	err := pool.Ping(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
}

func TestNewConnectionWithTimeout(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pool.Ping(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
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

	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	CloseConnection(pool)

	err := pool.Ping(ctx)
	if err == nil {
		t.Fatal("Expected Ping() to fail after pool was closed")
	}
}

func TestConnectionPoolProperties(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	err := pool.Ping(ctx)
	if err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	stats := pool.Stat()
	if stats.MaxConns() != 25 {
		t.Errorf("Expected MaxConns to be 25, got %d", stats.MaxConns())
	}
}
