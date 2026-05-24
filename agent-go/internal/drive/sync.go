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
	known := map[string]int64{}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			roots, err := s.ListSyncRoots(ctx)
			if err != nil {
				continue
			}
			for _, root := range roots {
				if !root.Enabled {
					continue
				}
				if known[root.ID] == root.UpdatedAt {
					continue
				}
				known[root.ID] = root.UpdatedAt
				go s.watchSingleRoot(ctx, root)
			}
		}
	}
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
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watcher.Events:
			if event.Name == "" || event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
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
			_ = s.importLocalFile(ctx, root, event.Name, info)
		case <-watcher.Errors:
		}
	}
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
	_, err := s.db.ExecContext(ctx, `UPDATE sync_roots SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().Unix(), id)
	return err
}

func (s *Service) importLocalFile(ctx context.Context, root SyncRoot, localPath string, info os.FileInfo) error {
	rel, err := filepath.Rel(root.LocalPath, localPath)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	var existing string
	err = s.db.QueryRowContext(ctx, `SELECT remote_file_id FROM sync_entries WHERE sync_root_id = ? AND local_path = ?`, root.ID, localPath).Scan(&existing)
	if err == nil && existing != "" {
		return nil
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
