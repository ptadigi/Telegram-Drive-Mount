package drive

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type SyncLogEntry struct {
	TS        string `json:"ts"`
	Level     string `json:"level"`
	Event     string `json:"event"`
	FileID    string `json:"file_id,omitempty"`
	FileName  string `json:"file_name,omitempty"`
	Phase     string `json:"phase,omitempty"`
	Size      int64  `json:"size,omitempty"`
	SyncState string `json:"sync_state,omitempty"`
	Error     string `json:"error,omitempty"`
	LocalPath string `json:"local_path,omitempty"`
}

func (s *Service) logSync(entry SyncLogEntry) {
	if entry.TS == "" {
		entry.TS = time.Now().UTC().Format(time.RFC3339)
	}
	if entry.Level == "" {
		entry.Level = "info"
	}
	logDir := filepath.Join(s.dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(logDir, "sync.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_ = json.NewEncoder(file).Encode(entry)
}

// LogSyncError is an exported helper for other packages (e.g. tus import) to
// record a sync error into sync.log without exposing the internal entry type.
func (s *Service) LogSyncError(event, fileName string, size int64, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	s.logSync(SyncLogEntry{Level: "error", Event: event, FileName: fileName, Size: size, Error: msg})
}

func (s *Service) RecentSyncLogs(ctx context.Context, limit int) ([]SyncLogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	path := filepath.Join(s.dataDir, "logs", "sync.log")
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SyncLogEntry{}, nil
		}
		return nil, err
	}
	defer file.Close()
	lines := make([]string, 0, limit)
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		line := scanner.Text()
		if len(lines) >= limit {
			copy(lines, lines[1:])
			lines[len(lines)-1] = line
		} else {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	entries := make([]SyncLogEntry, 0, len(lines))
	for i := len(lines) - 1; i >= 0; i-- {
		var entry SyncLogEntry
		if err := json.Unmarshal([]byte(lines[i]), &entry); err == nil {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}
