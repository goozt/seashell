package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// EncryptKey encrypts plaintext using AES-256-GCM with the given KEK.
// The returned ciphertext has the format: nonce(12 bytes) || ciphertext.
// If kek is empty, plaintext is returned unchanged (dev mode).
func EncryptKey(kek, plaintext []byte) ([]byte, error) {
	if len(kek) == 0 {
		return plaintext, nil
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptKey decrypts data produced by EncryptKey.
// If kek is empty, data is returned unchanged (dev mode).
func DecryptKey(kek, data []byte) ([]byte, error) {
	if len(kek) == 0 {
		return data, nil
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// ParseKEK decodes a 32-byte hex-encoded key encryption key.
// Returns nil if hexKey is empty (dev mode — no encryption).
func ParseKEK(hexKey string) ([]byte, error) {
	if hexKey == "" {
		return nil, nil
	}
	kek, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode KEY_ENCRYPTION_KEY: %w", err)
	}
	if len(kek) != 32 {
		return nil, fmt.Errorf("KEY_ENCRYPTION_KEY must be 32 bytes (64 hex chars), got %d bytes", len(kek))
	}
	return kek, nil
}
