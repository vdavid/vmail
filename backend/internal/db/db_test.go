package db

import (
	"context"
	"testing"
	"time"

	"github.com/vdavid/vmail/backend/internal/config"
)

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
