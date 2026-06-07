// Package secret provides at-rest encryption helpers for sensitive agent state.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	// KeySize is the AES-256 key size in bytes.
	KeySize = 32
	// EnvVar is the env name that holds the agent encryption key.
	EnvVar = "TD_AGENT_SESSION_KEY"
)

// LoadKey resolves the at-rest encryption key from the configured env variable.
// Accepts both hex (64 chars) and base64 (44 chars) encodings of 32 bytes.
// Returns nil key + nil error when the env var is empty (caller must decide
// whether encryption is mandatory).
func LoadKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(EnvVar))
	if raw == "" {
		return nil, nil
	}
	if key, err := hex.DecodeString(raw); err == nil && len(key) == KeySize {
		return key, nil
	}
	if key, err := base64.StdEncoding.DecodeString(raw); err == nil && len(key) == KeySize {
		return key, nil
	}
	if key, err := base64.RawStdEncoding.DecodeString(raw); err == nil && len(key) == KeySize {
		return key, nil
	}
	if key, err := base64.URLEncoding.DecodeString(raw); err == nil && len(key) == KeySize {
		return key, nil
	}
	return nil, fmt.Errorf("%s không hợp lệ: cần 32 byte (hex 64 ký tự hoặc base64)", EnvVar)
}

// Generate returns a fresh random 32-byte key encoded as hex.
// Useful for a one-shot CLI like `td-agent gen-key`.
func Generate() (string, error) {
	buf := make([]byte, KeySize)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Encrypt encrypts plaintext with AES-256-GCM. The output layout is:
//
//	"TDS1" magic (4 bytes) | nonce (12 bytes) | ciphertext+tag
//
// The magic prefix lets us detect already-encrypted blobs during migration.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, errors.New("invalid key size")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 0, 4+len(nonce)+len(plaintext)+gcm.Overhead())
	out = append(out, []byte("TDS1")...)
	out = append(out, nonce...)
	out = gcm.Seal(out, nonce, plaintext, nil)
	return out, nil
}

// Decrypt is the inverse of Encrypt. It returns ErrLegacyPlain when the input
// does not start with the "TDS1" magic so callers can treat the blob as
// legacy plaintext (e.g. an old Telegram session file).
func Decrypt(key, blob []byte) ([]byte, error) {
	if !HasMagic(blob) {
		return nil, ErrLegacyPlain
	}
	if len(key) != KeySize {
		return nil, errors.New("invalid key size")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(blob) < 4+nonceSize {
		return nil, errors.New("encrypted blob too short")
	}
	body := blob[4:]
	nonce := body[:nonceSize]
	cipherText := body[nonceSize:]
	return gcm.Open(nil, nonce, cipherText, nil)
}

// HasMagic reports whether the blob starts with the TDS1 magic prefix.
func HasMagic(blob []byte) bool {
	return len(blob) >= 4 && string(blob[:4]) == "TDS1"
}

// ErrLegacyPlain is returned by Decrypt when the input is not encrypted yet.
var ErrLegacyPlain = errors.New("plaintext (legacy)")
