package db

import (
	"context"
	"testing"

	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestGetOrCreateUser(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	t.Run("creates new user", func(t *testing.T) {
		email := "test@example.com"

		userID, err := GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("GetOrCreateUser failed: %v", err)
		}

		if userID == "" {
			t.Fatal("Expected non-empty user ID")
		}
	})

	t.Run("returns existing user", func(t *testing.T) {
		email := "existing@example.com"

		userID1, err := GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("First GetOrCreateUser failed: %v", err)
		}

		userID2, err := GetOrCreateUser(ctx, pool, email)
		if err != nil {
			t.Fatalf("Second GetOrCreateUser failed: %v", err)
		}

		if userID1 != userID2 {
			t.Errorf("Expected same user ID, got %s and %s", userID1, userID2)
		}
	})
}
