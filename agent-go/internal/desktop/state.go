package desktop

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// Mode describes how the desktop agent is configured to run.
type Mode string

const (
	ModeUnset  Mode = "unset"
	ModeLocal  Mode = "local"
	ModeRemote Mode = "remote"
)

// State is the persisted desktop onboarding configuration.
type State struct {
	Mode       string `json:"mode"`
	ServerURL  string `json:"server_url,omitempty"`
	DeviceName string `json:"device_name,omitempty"`
	MountPoint string `json:"mount_point,omitempty"`
	UpdatedAt  int64  `json:"updated_at,omitempty"`
}

// Store persists desktop state to a JSON file under the data dir.
type Store struct {
	path string
	mu   sync.RWMutex
}

func NewStore(dataDir string) *Store {
	return &Store{path: filepath.Join(dataDir, "desktop.json")}
}

func (s *Store) Load() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return State{Mode: string(ModeUnset)}
	}
	var st State
	if err := json.Unmarshal(raw, &st); err != nil || st.Mode == "" {
		return State{Mode: string(ModeUnset)}
	}
	return st
}

func (s *Store) Save(st State) error {
	if st.Mode == "" {
		return errors.New("mode trống")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(&st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
