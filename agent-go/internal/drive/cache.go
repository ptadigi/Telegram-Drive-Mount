package drive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type CacheStats struct {
	Mode      string `json:"mode"`
	MaxBytes  int64  `json:"max_bytes"`
	UsedBytes int64  `json:"used_bytes"`
	Files     int    `json:"files"`
}

type CachePolicy struct {
	Mode     string
	MaxBytes int64
}

func (s *Service) SetCachePolicy(mode string, maxBytes int64) {
	if mode == "" {
		mode = "smart"
	}
	s.cacheMu.Lock()
	s.cachePolicy = CachePolicy{Mode: mode, MaxBytes: maxBytes}
	s.cacheMu.Unlock()
}

func (s *Service) GetCachePolicy() CachePolicy {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return s.cachePolicy
}

func (s *Service) CacheStats(ctx context.Context) (CacheStats, error) {
	policy := s.GetCachePolicy()
	rows, err := s.db.QueryContext(ctx, `SELECT COALESCE(local_path, ''), size FROM files WHERE deleted_at IS NULL`)
	if err != nil {
		return CacheStats{}, err
	}
	defer rows.Close()
	var used int64
	count := 0
	for rows.Next() {
		var path string
		var size int64
		if err := rows.Scan(&path, &size); err != nil {
			return CacheStats{}, err
		}
		if path == "" {
			continue
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			continue
		}
		used += info.Size()
		count++
	}
	return CacheStats{Mode: policy.Mode, MaxBytes: policy.MaxBytes, UsedBytes: used, Files: count}, rows.Err()
}

type cacheItem struct {
	id       string
	path     string
	size     int64
	accessed int64
	state    string
}

func (s *Service) CleanupCache(ctx context.Context) (int, error) {
	policy := s.GetCachePolicy()
	if policy.Mode == "mirror" {
		return 0, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, COALESCE(local_path, ''), size, COALESCE(last_accessed_at, updated_at), sync_state FROM files WHERE deleted_at IS NULL AND COALESCE(local_path, '') <> ''`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	items := make([]cacheItem, 0)
	for rows.Next() {
		var item cacheItem
		if err := rows.Scan(&item.id, &item.path, &item.size, &item.accessed, &item.state); err != nil {
			return 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if policy.Mode == "cloud_only" {
		return s.removeCacheItems(ctx, items, func(item cacheItem) bool { return item.state == "telegram_synced" }), nil
	}
	if policy.Mode == "smart" {
		var used int64
		for _, item := range items {
			used += item.size
		}
		if policy.MaxBytes <= 0 || used <= policy.MaxBytes {
			return 0, nil
		}
		sort.Slice(items, func(i, j int) bool { return items[i].accessed < items[j].accessed })
		removed := 0
		for _, item := range items {
			if used <= policy.MaxBytes {
				break
			}
			if item.state != "telegram_synced" {
				continue
			}
			if err := s.evictCacheItem(ctx, item); err != nil {
				continue
			}
			used -= item.size
			removed++
		}
		return removed, nil
	}
	return 0, nil
}

func (s *Service) removeCacheItems(ctx context.Context, items []cacheItem, predicate func(cacheItem) bool) int {
	removed := 0
	for _, item := range items {
		if !predicate(item) {
			continue
		}
		if err := s.evictCacheItem(ctx, item); err != nil {
			continue
		}
		removed++
	}
	return removed
}

func (s *Service) evictCacheItem(ctx context.Context, item cacheItem) error {
	if item.path == "" {
		return nil
	}
	if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE files SET local_path = '', updated_at = ? WHERE id = ?`, time.Now().Unix(), item.id); err != nil {
		return err
	}
	return nil
}

func (s *Service) CacheWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.CleanupCache(ctx)
		}
	}
}

func (s *Service) RecordFileAccess(ctx context.Context, fileID string) {
	if fileID == "" {
		return
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE files SET last_accessed_at = ?, updated_at = ? WHERE id = ?`, time.Now().Unix(), time.Now().Unix(), fileID)
}

func (s *Service) localCachePath(file File) string {
	if file.LocalPath != "" {
		return file.LocalPath
	}
	return filepath.Join(s.dataDir, "uploads", fmt.Sprintf("%s-%s", file.ID, filepath.Base(file.Name)))
}
