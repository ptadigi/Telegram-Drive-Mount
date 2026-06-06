package drive

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type SyncRoot struct {
	ID             string `json:"id"`
	LocalPath      string `json:"local_path"`
	RemoteFolderID string `json:"remote_folder_id,omitempty"`
	Mode           string `json:"mode"`
	Enabled        bool   `json:"enabled"`
	Status         string `json:"status"`
	LastScanAt     int64  `json:"last_scan_at"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type CreateSyncRootInput struct {
	LocalPath      string `json:"local_path"`
	RemoteFolderID string `json:"remote_folder_id"`
	Mode           string `json:"mode"`
}

type UpdateSyncRootInput struct {
	ID      string `json:"id"`
	Enabled *bool  `json:"enabled"`
}

func (s *Service) ListSyncRoots(ctx context.Context) ([]SyncRoot, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, local_path, COALESCE(remote_folder_id, ''), mode, enabled, COALESCE(status, 'idle'), COALESCE(last_scan_at, 0), created_at, updated_at
		FROM sync_roots
		WHERE enabled IN (0, 1)
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("đọc danh sách thư mục đồng bộ: %w", err)
	}
	defer rows.Close()
	roots := make([]SyncRoot, 0)
	for rows.Next() {
		var root SyncRoot
		var enabled int
		if err := rows.Scan(&root.ID, &root.LocalPath, &root.RemoteFolderID, &root.Mode, &enabled, &root.Status, &root.LastScanAt, &root.CreatedAt, &root.UpdatedAt); err != nil {
			return nil, fmt.Errorf("đọc thư mục đồng bộ: %w", err)
		}
		root.Enabled = enabled == 1
		roots = append(roots, root)
	}
	return roots, rows.Err()
}

func (s *Service) CreateSyncRoot(ctx context.Context, input CreateSyncRootInput) (SyncRoot, error) {
	localPath := strings.TrimSpace(input.LocalPath)
	if localPath == "" {
		return SyncRoot{}, fmt.Errorf("vui lòng nhập đường dẫn thư mục local")
	}
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return SyncRoot{}, fmt.Errorf("chuẩn hóa đường dẫn sync: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return SyncRoot{}, fmt.Errorf("không đọc được thư mục sync: %w", err)
	}
	if !info.IsDir() {
		return SyncRoot{}, fmt.Errorf("đường dẫn sync phải là thư mục")
	}
	if input.RemoteFolderID != "" {
		if err := s.ensureFolderExists(ctx, input.RemoteFolderID); err != nil {
			return SyncRoot{}, err
		}
	}
	mode := input.Mode
	if mode == "" {
		mode = "upload_only"
	}
	now := time.Now().Unix()
	root := SyncRoot{ID: newID(), LocalPath: absPath, RemoteFolderID: input.RemoteFolderID, Mode: mode, Enabled: true, Status: "scanning", CreatedAt: now, UpdatedAt: now}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO sync_roots (id, local_path, remote_folder_id, mode, enabled, status, last_scan_at, created_at, updated_at)
		VALUES (?, ?, NULLIF(?, ''), ?, 1, ?, 0, ?, ?)
	`, root.ID, root.LocalPath, root.RemoteFolderID, root.Mode, root.Status, now, now)
	if err == nil {
		s.events.Publish("syncroot.created", root)
	}
	if err != nil {
		return SyncRoot{}, fmt.Errorf("tạo sync root: %w", err)
	}
	if err := s.ScanSyncRoot(ctx, root.ID); err != nil {
		_ = s.setSyncRootStatus(ctx, root.ID, "error")
		return root, err
	}
	return root, nil
}

func (s *Service) UpdateSyncRoot(ctx context.Context, input UpdateSyncRootInput) error {
	if input.ID == "" {
		return fmt.Errorf("thiếu id thư mục đồng bộ")
	}
	if input.Enabled == nil {
		return nil
	}
	enabled := 0
	status := "paused"
	if *input.Enabled {
		enabled = 1
		status = "watching"
	}
	_, err := s.db.ExecContext(ctx, `UPDATE sync_roots SET enabled = ?, status = ?, updated_at = ? WHERE id = ?`, enabled, status, time.Now().Unix(), input.ID)
	return err
}

func (s *Service) DeleteSyncRoot(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("thiếu id thư mục đồng bộ")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE sync_roots SET enabled = -1, status = 'removed', updated_at = ? WHERE id = ?`, time.Now().Unix(), id)
	return err
}

func (s *Service) ScanSyncRoot(ctx context.Context, id string) error {
	root, err := s.getSyncRoot(ctx, id)
	if err != nil {
		return err
	}
	if !root.Enabled {
		return fmt.Errorf("thư mục đồng bộ đang tắt")
	}
	if err := s.setSyncRootStatus(ctx, root.ID, "scanning"); err != nil {
		return err
	}
	err = filepath.WalkDir(root.LocalPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return s.importLocalFile(ctx, root, path, info)
	})
	if err != nil {
		_ = s.setSyncRootStatus(ctx, root.ID, "error")
		return fmt.Errorf("quét thư mục sync: %w", err)
	}
	now := time.Now().Unix()
	_, err = s.db.ExecContext(ctx, `UPDATE sync_roots SET status = 'watching', last_scan_at = ?, updated_at = ? WHERE id = ?`, now, now, root.ID)
	return err
}

func (s *Service) SyncRootWatcher(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.cancelAllSyncWatchers()
			return
		case <-ticker.C:
			roots, err := s.ListSyncRoots(ctx)
			if err != nil {
				continue
			}
			active := map[string]struct{}{}
			for _, root := range roots {
				if !root.Enabled {
					s.cancelSyncWatcher(root.ID)
					continue
				}
				active[root.ID] = struct{}{}
				if !s.startSyncWatcher(ctx, root) {
					continue
				}
			}
			for id := range s.copySyncWatcherIDs() {
				if _, ok := active[id]; !ok {
					s.cancelSyncWatcher(id)
				}
			}
		}
	}
}

func (s *Service) startSyncWatcher(parent context.Context, root SyncRoot) bool {
	s.syncWatchMu.Lock()
	if s.syncWatchers == nil {
		s.syncWatchers = map[string]syncWatcherEntry{}
	}
	if entry, ok := s.syncWatchers[root.ID]; ok {
		if entry.updatedAt == root.UpdatedAt {
			s.syncWatchMu.Unlock()
			return false
		}
		entry.cancel()
		delete(s.syncWatchers, root.ID)
	}
	ctx, cancel := context.WithCancel(parent)
	s.syncWatchers[root.ID] = syncWatcherEntry{cancel: cancel, updatedAt: root.UpdatedAt}
	s.syncWatchMu.Unlock()
	go func() {
		s.watchSingleRoot(ctx, root)
		s.syncWatchMu.Lock()
		if entry, ok := s.syncWatchers[root.ID]; ok && entry.updatedAt == root.UpdatedAt {
			delete(s.syncWatchers, root.ID)
		}
		s.syncWatchMu.Unlock()
	}()
	return true
}

func (s *Service) cancelSyncWatcher(rootID string) {
	s.syncWatchMu.Lock()
	if entry, ok := s.syncWatchers[rootID]; ok {
		entry.cancel()
		delete(s.syncWatchers, rootID)
	}
	s.syncWatchMu.Unlock()
}

func (s *Service) cancelAllSyncWatchers() {
	s.syncWatchMu.Lock()
	for id, entry := range s.syncWatchers {
		entry.cancel()
		delete(s.syncWatchers, id)
	}
	s.syncWatchMu.Unlock()
}

func (s *Service) copySyncWatcherIDs() map[string]struct{} {
	s.syncWatchMu.Lock()
	defer s.syncWatchMu.Unlock()
	out := make(map[string]struct{}, len(s.syncWatchers))
	for id := range s.syncWatchers {
		out[id] = struct{}{}
	}
	return out
}

func (s *Service) watchSingleRoot(ctx context.Context, root SyncRoot) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()
	_ = filepath.WalkDir(root.LocalPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr == nil && entry.IsDir() {
			_ = watcher.Add(path)
		}
		return nil
	})
	pending := map[string]*pendingChange{}
	debounce := time.NewTicker(2 * time.Second)
	defer debounce.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Name == "" {
				continue
			}
			if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				s.handleSyncRemoval(ctx, root, event.Name)
				delete(pending, event.Name)
				continue
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}
			info, err := os.Stat(event.Name)
			if err != nil {
				continue
			}
			if info.IsDir() {
				_ = watcher.Add(event.Name)
				continue
			}
			pending[event.Name] = &pendingChange{path: event.Name, size: info.Size(), mtime: info.ModTime().Unix(), ready: 0}
		case <-debounce.C:
			for path, change := range pending {
				info, err := os.Stat(path)
				if err != nil {
					delete(pending, path)
					continue
				}
				if info.IsDir() {
					delete(pending, path)
					continue
				}
				if info.Size() == change.size && info.ModTime().Unix() == change.mtime {
					change.ready++
					if change.ready >= 1 {
						_ = s.importLocalFile(ctx, root, path, info)
						delete(pending, path)
					}
					continue
				}
				change.size = info.Size()
				change.mtime = info.ModTime().Unix()
				change.ready = 0
			}
		case <-watcher.Errors:
		}
	}
}

type pendingChange struct {
	path  string
	size  int64
	mtime int64
	ready int
}

func (s *Service) handleSyncRemoval(ctx context.Context, root SyncRoot, localPath string) {
	var remoteID string
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(remote_file_id, '') FROM sync_entries WHERE sync_root_id = ? AND local_path = ?`, root.ID, localPath).Scan(&remoteID); err != nil {
		return
	}
	if remoteID == "" {
		return
	}
	now := time.Now().Unix()
	_, _ = s.db.ExecContext(ctx, `UPDATE files SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, remoteID)
	_, _ = s.db.ExecContext(ctx, `UPDATE sync_entries SET state = 'pending_delete', last_error = NULL WHERE sync_root_id = ? AND local_path = ?`, root.ID, localPath)
	s.events.Publish("file.trashed", map[string]any{"id": remoteID, "deleted_at": now})
}

func (s *Service) getSyncRoot(ctx context.Context, id string) (SyncRoot, error) {
	var root SyncRoot
	var enabled int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, local_path, COALESCE(remote_folder_id, ''), mode, enabled, COALESCE(status, 'idle'), COALESCE(last_scan_at, 0), created_at, updated_at
		FROM sync_roots WHERE id = ? AND enabled IN (0, 1)
	`, id).Scan(&root.ID, &root.LocalPath, &root.RemoteFolderID, &root.Mode, &enabled, &root.Status, &root.LastScanAt, &root.CreatedAt, &root.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return SyncRoot{}, fmt.Errorf("không tìm thấy thư mục đồng bộ")
		}
		return SyncRoot{}, err
	}
	root.Enabled = enabled == 1
	return root, nil
}

func (s *Service) setSyncRootStatus(ctx context.Context, id string, status string) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `UPDATE sync_roots SET status = ?, updated_at = ? WHERE id = ?`, status, now, id)
	if err == nil {
		s.events.Publish("syncroot.updated", map[string]any{"id": id, "status": status, "updated_at": now})
	}
	return err
}

func (s *Service) importLocalFile(ctx context.Context, root SyncRoot, localPath string, info os.FileInfo) error {
	rel, err := filepath.Rel(root.LocalPath, localPath)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	var existing string
	var existingMtime sql.NullInt64
	err = s.db.QueryRowContext(ctx, `SELECT remote_file_id, COALESCE(local_mtime, 0) FROM sync_entries WHERE sync_root_id = ? AND local_path = ?`, root.ID, localPath).Scan(&existing, &existingMtime)
	if err == nil && existing != "" {
		if existingMtime.Valid && existingMtime.Int64 == info.ModTime().Unix() {
			return nil
		}
	}
	file, err := s.SaveLocalFile(ctx, localPath, root.RemoteFolderID, rel)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO sync_entries (id, sync_root_id, local_path, remote_file_id, local_mtime, state)
		VALUES (?, ?, ?, ?, ?, 'pending_upload')
	`, newID(), root.ID, localPath, file.ID, info.ModTime().Unix())
	return err
}
