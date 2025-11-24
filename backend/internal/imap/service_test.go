package imap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vdavid/vmail/backend/internal/db"
	"github.com/vdavid/vmail/backend/internal/testutil"
)

// mockThreadCountUpdater is a mock implementation of ThreadCountUpdater for testing.
type mockThreadCountUpdater struct {
	mock.Mock
}

func (m *mockThreadCountUpdater) UpdateThreadCount(ctx context.Context, userID, folderName string) error {
	args := m.Called(ctx, userID, folderName)
	return args.Error(0)
}

// TestShouldSyncFolder tests the cache TTL logic using a real database.
// This is an integration test that verifies the ShouldSyncFolder logic works correctly.
func TestShouldSyncFolder(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS folder_sync_timestamps (
			user_id     UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			folder_name TEXT        NOT NULL,
			synced_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (user_id, folder_name)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to ensure folder_sync_timestamps table exists: %v", err)
	}

	encryptor := testutil.GetTestEncryptor(t)
	service := NewService(pool, NewPool(), encryptor)
	defer service.Close()

	userID, err := db.GetOrCreateUser(ctx, pool, "sync-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	folderName := "INBOX"

	tests := []struct {
		name        string
		setup       func()
		expected    bool
		description string
	}{
		{
			name:        "returns true when no sync timestamp exists",
			setup:       func() {}, // No setup needed
			expected:    true,
			description: "should sync when no timestamp exists",
		},
		{
			name: "returns false when cache is fresh",
			setup: func() {
				_ = db.SetFolderSyncInfo(ctx, pool, userID, folderName, nil)
			},
			expected:    false,
			description: "should not sync when cache is fresh",
		},
		{
			name: "returns true when cache is stale",
			setup: func() {
				_, _ = pool.Exec(ctx, `
					UPDATE folder_sync_timestamps 
					SET synced_at = $1 
					WHERE user_id = $2 AND folder_name = $3
				`, time.Now().Add(-10*time.Minute), userID, folderName)
			},
			expected:    true,
			description: "should sync when cache is stale (older than 5 minutes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			shouldSync, err := service.ShouldSyncFolder(ctx, userID, folderName)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, shouldSync, tt.description)
		})
	}
}

func TestGetFolderSyncInfoWithUID(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS folder_sync_timestamps (
			user_id        UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			folder_name    TEXT        NOT NULL,
			synced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_synced_uid BIGINT,
			thread_count   INT DEFAULT 0,
			PRIMARY KEY (user_id, folder_name)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to ensure folder_sync_timestamps table exists: %v", err)
	}

	userID, err := db.GetOrCreateUser(ctx, pool, "uid-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	tests := []struct {
		name        string
		folderName  string
		setUID      *int64
		expectedUID *int64
		expectedNil bool
	}{
		{
			name:        "returns UID when set",
			folderName:  "INBOX",
			setUID:      intPtr(50000),
			expectedUID: intPtr(50000),
			expectedNil: false,
		},
		{
			name:        "returns nil UID when not set",
			folderName:  "TestFolder",
			setUID:      nil,
			expectedUID: nil,
			expectedNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.SetFolderSyncInfo(ctx, pool, userID, tt.folderName, tt.setUID)
			assert.NoError(t, err)

			info, err := db.GetFolderSyncInfo(ctx, pool, userID, tt.folderName)
			assert.NoError(t, err)
			assert.NotNil(t, info)

			if tt.expectedNil {
				assert.Nil(t, info.LastSyncedUID)
			} else {
				assert.NotNil(t, info.LastSyncedUID)
				assert.Equal(t, *tt.expectedUID, *info.LastSyncedUID)
			}
		})
	}
}

func TestService_updateThreadCountInBackground(t *testing.T) {
	pool := testutil.NewTestDB(t)
	defer pool.Close()

	encryptor := testutil.GetTestEncryptor(t)
	mockUpdater := &mockThreadCountUpdater{}
	service := NewServiceWithThreadCountUpdater(pool, NewPool(), encryptor, mockUpdater)
	defer service.Close()

	ctx := context.Background()
	userID, err := db.GetOrCreateUser(ctx, pool, "thread-count-test@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	folderName := "INBOX"

	tests := []struct {
		name           string
		setupMock      func()
		testUserID     string
		testFolderName string
		expectError    bool
		waitTime       time.Duration
	}{
		{
			name: "succeeds with valid database connection",
			setupMock: func() {
				mockUpdater.On("UpdateThreadCount", mock.Anything, userID, folderName).
					Return(nil).
					Once()
			},
			testUserID:     userID,
			testFolderName: folderName,
			expectError:    false,
			waitTime:       100 * time.Millisecond,
		},
		{
			name: "handles database error gracefully",
			setupMock: func() {
				mockUpdater.On("UpdateThreadCount", mock.Anything, "00000000-0000-0000-0000-000000000000", "NonExistentFolder").
					Return(errors.New("database error")).
					Once()
			},
			testUserID:     "00000000-0000-0000-0000-000000000000",
			testFolderName: "NonExistentFolder",
			expectError:    true,
			waitTime:       200 * time.Millisecond,
		},
		{
			name: "handles context timeout",
			setupMock: func() {
				mockUpdater.On("UpdateThreadCount", mock.Anything, userID, folderName).
					Return(context.DeadlineExceeded).
					Once()
			},
			testUserID:     userID,
			testFolderName: folderName,
			expectError:    true,
			waitTime:       200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock for each test
			mockUpdater.ExpectedCalls = nil
			mockUpdater.Calls = nil
			tt.setupMock()

			service.updateThreadCountInBackground(tt.testUserID, tt.testFolderName)

			// Give the goroutine time to complete
			time.Sleep(tt.waitTime)

			// Verify mock expectations were met
			mockUpdater.AssertExpectations(t)

			// Test should complete without panicking
			// The function logs errors but doesn't panic, which is the expected behavior
		})
	}
}

func intPtr(i int64) *int64 {
	return &i
}
