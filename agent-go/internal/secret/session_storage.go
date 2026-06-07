package secret

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gotd/td/session"
)

// EncryptedSessionStorage wraps gotd's session.Loader so the on-disk Telegram
// session file is encrypted with AES-256-GCM using a key from the environment
// variable. Layout on disk: TDS1 magic | 12-byte nonce | ciphertext+tag.
type EncryptedSessionStorage struct {
	Path string
	Key  []byte
}

// LoadSession reads + decrypts the session file. Returns nil, nil when no file
// is present (gotd will then start a fresh session). Migrates plaintext files
// transparently the first time they are read with a key configured.
func (s *EncryptedSessionStorage) LoadSession(_ context.Context) ([]byte, error) {
	if s == nil || s.Path == "" {
		return nil, errors.New("encrypted session storage thiếu Path")
	}
	if len(s.Key) != KeySize {
		return nil, errors.New("encrypted session storage thiếu key 32 byte")
	}
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if HasMagic(raw) {
		return Decrypt(s.Key, raw)
	}
	// Legacy plaintext: re-encrypt in place so future reads are protected.
	if err := s.StoreSession(context.Background(), raw); err != nil {
		return nil, fmt.Errorf("migrate session sang dạng mã hoá: %w", err)
	}
	return raw, nil
}

// StoreSession encrypts data with AES-256-GCM and writes atomically.
func (s *EncryptedSessionStorage) StoreSession(_ context.Context, data []byte) error {
	if s == nil || s.Path == "" {
		return errors.New("encrypted session storage thiếu Path")
	}
	if len(s.Key) != KeySize {
		return errors.New("encrypted session storage thiếu key 32 byte")
	}
	enc, err := Encrypt(s.Key, data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.Path), filepath.Base(s.Path)+".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(enc); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.Path)
}

// Compile-time interface check against gotd's session.Storage shape.
var _ session.Storage = (*EncryptedSessionStorage)(nil)
