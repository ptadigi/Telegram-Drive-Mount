// Package remote implements the agent client that talks to a Telegram Drive
// VPS over HTTPS, instead of running the local SQLite + Telegram stack.
package remote

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Token struct {
	BaseURL  string `json:"base_url"`
	Token    string `json:"token"`
	DeviceID string `json:"device_id"`
	UserID   string `json:"user_id"`
	IssuedAt int64  `json:"issued_at"`
}

func DefaultTokenPath() string {
	if env := strings.TrimSpace(os.Getenv("TD_AGENT_TOKEN_FILE")); env != "" {
		return env
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "TelegramVirtualDrive", "agent-client", "token.json")
}

func SaveToken(path string, tok Token) error {
	if path == "" {
		return errors.New("thieu duong dan token")
	}
	if tok.Token == "" || tok.BaseURL == "" {
		return errors.New("token rong, khong luu")
	}
	if tok.IssuedAt == 0 {
		tok.IssuedAt = time.Now().Unix()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmp.Name(), 0o600)
	}
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&tok); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmp.Name(), 0o600)
	}
	return os.Rename(tmp.Name(), path)
}

func LoadToken(path string) (Token, error) {
	if path == "" {
		return Token{}, errors.New("thieu duong dan token")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Token{}, err
	}
	var tok Token
	if err := json.Unmarshal(raw, &tok); err != nil {
		return Token{}, fmt.Errorf("token file khong hop le: %w", err)
	}
	if tok.Token == "" || tok.BaseURL == "" {
		return Token{}, errors.New("token file thieu base_url hoac token")
	}
	return tok, nil
}

func DeleteToken(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}