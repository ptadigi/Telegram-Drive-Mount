package drive

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func (s *Service) BackupMetadata(ctx context.Context) (string, error) {
	dir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	source := filepath.Join(s.dataDir, "metadata.db")
	if _, err := os.Stat(source); err != nil {
		return "", fmt.Errorf("không tìm thấy metadata.db: %w", err)
	}
	target := filepath.Join(dir, fmt.Sprintf("metadata-%s.db", time.Now().UTC().Format("20060102-150405")))
	src, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	if err := pruneBackups(dir, 14); err != nil {
		return target, err
	}
	return target, nil
}

func (s *Service) BackupWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if path, err := s.BackupMetadata(ctx); err == nil {
				s.events.Publish("metadata.backup", map[string]any{"path": path, "ts": time.Now().Unix()})
			}
		}
	}
}

func pruneBackups(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type backupFile struct {
		path    string
		modTime time.Time
	}
	files := make([]backupFile, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, backupFile{path: filepath.Join(dir, entry.Name()), modTime: info.ModTime()})
	}
	if len(files) <= keep {
		return nil
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modTime.After(files[j].modTime) })
	for _, file := range files[keep:] {
		_ = os.Remove(file.path)
	}
	return nil
}
