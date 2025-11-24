package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

func TestGetOrCreateUser(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	tests := []struct {
		name        string
		email       string
		expectNew   bool
		checkResult func(*testing.T, string, string)
	}{
		{
			name:      "creates new user",
			email:     "test@example.com",
			expectNew: true,
			checkResult: func(t *testing.T, userID1, userID2 string) {
				assert.NotEmpty(t, userID1)
			},
		},
		{
			name:      "returns existing user",
			email:     "existing@example.com",
			expectNew: false,
			checkResult: func(t *testing.T, userID1, userID2 string) {
				assert.Equal(t, userID1, userID2, "should return same user ID for same email")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID1, err := GetOrCreateUser(ctx, pool, tt.email)
			assert.NoError(t, err)
			assert.NotEmpty(t, userID1)

			if !tt.expectNew {
				userID2, err := GetOrCreateUser(ctx, pool, tt.email)
				assert.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, userID1, userID2)
				}
			} else {
				if tt.checkResult != nil {
					tt.checkResult(t, userID1, "")
				}
			}
		})
	}
}
