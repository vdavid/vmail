package testutil

import (
	"encoding/base64"
	"testing"

	"github.com/vdavid/vmail/backend/internal/crypto"
)

// TestEncryptionKeyBase64 is a base64-encoded 32-byte encryption key used for testing.
// This key is shared across all test packages to ensure consistent test behavior.
// The key itself is "test-key-12345678901234567890123".
const TestEncryptionKeyBase64 = "dGVzdC1rZXktMTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM="

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
