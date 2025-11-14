# Crypto

The `crypto` package provides encryption and decryption functionality for sensitive data like user passwords.

## Components

* **`internal/crypto/encryption.go`**: AES-GCM encryption implementation.
    * `Encryptor`: Struct holding the encryption key.
    * `NewEncryptor`: Creates a new encryptor from a base64-encoded 32-byte key.
    * `Encrypt`: Encrypts plaintext using AES-GCM with a random nonce.
    * `Decrypt`: Decrypts ciphertext, verifying authenticity and integrity.

## Encryption scheme

* **Algorithm:** AES-256-GCM (Galois/Counter Mode)
* **Key size:** 32 bytes (256 bits)
* **Nonce:** Randomly generated for each encryption (12 bytes for GCM)
* **Ciphertext format:** `[nonce][encrypted_data][auth_tag]`
    * The nonce is prepended to the ciphertext for use during decryption.
    * The authentication tag is appended by GCM to verify data integrity.

## Security properties

* **Confidentiality:** Data is encrypted and cannot be read without the key.
* **Authenticity:** GCM provides authentication, detecting tampering or corruption.
* **Nonce uniqueness:** Each encryption uses a random nonce, ensuring the same plaintext produces different ciphertexts.
* **Key storage:** The encryption key is stored in memory as plain bytes (standard practice for application-level encryption).

## Usage

* Used to encrypt/decrypt IMAP and SMTP passwords before storing them in the database.
* The encryption key is provided via the `VMAIL_ENCRYPTION_KEY_BASE64` environment variable.
* The same key must be used across all application instances to decrypt previously encrypted data.
