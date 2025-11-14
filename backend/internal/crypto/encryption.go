package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// Encryptor provides encryption and decryption functionality using AES-GCM (Galois/Counter Mode).
// AES-GCM provides both confidentiality and authenticity, making it suitable for encrypting
// sensitive data like user passwords. The key is stored in memory as plain bytes.
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new Encryptor with the given key.
func NewEncryptor(base64Key string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (256 bits), got %d bytes", len(key))
	}

	return &Encryptor{key: key}, nil
}

// Encrypt encrypts the given plaintext using AES-GCM.
// The returned ciphertext format is: [nonce][encrypted_data][auth_tag]
// where the nonce is prepended to the ciphertext for use during decryption.
// Each encryption uses a random nonce, ensuring the same plaintext produces different ciphertexts.
func (e *Encryptor) Encrypt(plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

// Decrypt decrypts the given ciphertext using AES-GCM.
// The ciphertext format is expected to be: [nonce][encrypted_data][auth_tag]
// where the nonce is prepended. Returns an error if the ciphertext is invalid,
// corrupted, or was encrypted with a different key (authentication failure).
func (e *Encryptor) Decrypt(ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
