package crypto

import (
	"encoding/base64"
	"testing"
)

func TestNewEncryptor(t *testing.T) {
	t.Run("valid 32-byte key", func(t *testing.T) {
		key := make([]byte, 32)
		base64Key := base64.StdEncoding.EncodeToString(key)

		encryptor, err := NewEncryptor(base64Key)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if encryptor == nil {
			t.Fatal("Expected encryptor, got nil")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := NewEncryptor("not-valid-base64!!!")
		if err == nil {
			t.Fatal("Expected error for invalid base64, got nil")
		}
	})

	t.Run("wrong key length", func(t *testing.T) {
		key := make([]byte, 16)
		base64Key := base64.StdEncoding.EncodeToString(key)

		_, err := NewEncryptor(base64Key)
		if err == nil {
			t.Fatal("Expected error for wrong key length, got nil")
		}
	})
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, err := NewEncryptor(base64Key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"simple password", "mypassword123"},
		{"complex password", "P@ssw0rd!#$%^&*()"},
		{"empty string", ""},
		{"unicode", "пароль密码🔐"},
		{"long text", "This is a very long password with many characters to test the encryption and decryption of longer strings"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ciphertext, err := encryptor.Encrypt(tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			if len(ciphertext) == 0 {
				t.Fatal("Expected non-empty ciphertext")
			}

			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("Expected %q, got %q", tc.plaintext, decrypted)
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	key := make([]byte, 32)
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, err := NewEncryptor(base64Key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := "same password"

	ciphertext1, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("First encrypt failed: %v", err)
	}

	ciphertext2, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Second encrypt failed: %v", err)
	}

	if string(ciphertext1) == string(ciphertext2) {
		t.Error("Expected different ciphertexts for same plaintext (nonce should be different)")
	}

	decrypted1, _ := encryptor.Decrypt(ciphertext1)
	decrypted2, _ := encryptor.Decrypt(ciphertext2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both ciphertexts should decrypt to the same plaintext")
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	base64Key := base64.StdEncoding.EncodeToString(key)

	encryptor, err := NewEncryptor(base64Key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	t.Run("too short", func(t *testing.T) {
		_, err := encryptor.Decrypt([]byte("short"))
		if err == nil {
			t.Error("Expected error for too short ciphertext, got nil")
		}
	})

	t.Run("corrupted data", func(t *testing.T) {
		ciphertext, _ := encryptor.Encrypt("test")
		ciphertext[len(ciphertext)-1] ^= 0xFF

		_, err := encryptor.Decrypt(ciphertext)
		if err == nil {
			t.Error("Expected error for corrupted ciphertext, got nil")
		}
	})
}
