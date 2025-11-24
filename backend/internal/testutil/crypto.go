package testutil

import (
	"encoding/base64"
	"testing"

	"github.com/vdavid/vmail/backend/internal/crypto"
)

// GetTestEncryptor creates a test encryptor with a deterministic key for testing.
// This is shared across all test packages to avoid duplication.
func GetTestEncryptor(t *testing.T) *crypto.Encryptor {
	t.Helper()

	// Use the same test key pattern as api package tests
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, err := crypto.NewEncryptor(base64Key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}
	return encryptor
}
