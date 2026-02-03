package network

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// CredentialCrypto handles AES-256-GCM encryption for network credentials.
type CredentialCrypto struct {
	key []byte // 32 bytes derived from SESSION_SECRET
}

// NewCredentialCrypto creates a new crypto instance.
// The secret should be the SESSION_SECRET from config.
func NewCredentialCrypto(secret string) *CredentialCrypto {
	// Derive 32-byte key using SHA-256
	hash := sha256.Sum256([]byte(secret))
	return &CredentialCrypto{key: hash[:]}
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext with prepended nonce.
// Empty strings return empty (no encryption needed).
func (c *CredentialCrypto) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM.
// Empty strings return empty (no decryption needed).
func (c *CredentialCrypto) Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
