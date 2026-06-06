package drive

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type retryEntry struct {
	attempts int
	notBefore time.Time
}

var (
	retryMu      sync.Mutex
	retryState   = map[string]*retryEntry{}
	floodWaitRe  = regexp.MustCompile(`FLOOD_WAIT_(\d+)`)
	backoffSteps = []time.Duration{30 * time.Second, 2 * time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}
)

func (s *Service) uploadGateAllow(ctx context.Context, fileID string) bool {
	retryMu.Lock()
	entry, ok := retryState[fileID]
	retryMu.Unlock()
	if !ok {
		return true
	}
	return time.Now().After(entry.notBefore)
}

func (s *Service) scheduleRetry(fileID string, err error) {
	if fileID == "" {
		return
	}
	retryMu.Lock()
	defer retryMu.Unlock()
	entry, ok := retryState[fileID]
	if !ok {
		entry = &retryEntry{}
		retryState[fileID] = entry
	}
	entry.attempts++
	wait := backoffFor(entry.attempts, err)
	entry.notBefore = time.Now().Add(wait)
}

func (s *Service) clearRetry(fileID string) {
	retryMu.Lock()
	defer retryMu.Unlock()
	delete(retryState, fileID)
}

func backoffFor(attempts int, err error) time.Duration {
	if err != nil {
		if match := floodWaitRe.FindStringSubmatch(err.Error()); len(match) == 2 {
			if seconds, parseErr := strconv.Atoi(match[1]); parseErr == nil && seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
		}
		if strings.Contains(err.Error(), "context canceled") {
			return time.Second
		}
	}
	idx := attempts - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(backoffSteps) {
		idx = len(backoffSteps) - 1
	}
	return backoffSteps[idx]
}

func (s *Service) maybeEvictAfterSync(ctx context.Context, fileID string) {
	policy := s.GetCachePolicy()
	if policy.Mode != "cloud_only" {
		return
	}
	var path, origin string
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(local_path, ''), COALESCE(cache_origin, 'upload_cache') FROM files WHERE id = ?`, fileID).Scan(&path, &origin); err != nil {
		return
	}
	if origin == CacheOriginSync {
		return
	}
	if !s.pathInsideCache(path) {
		return
	}
	_ = s.evictCacheItem(ctx, cacheItem{id: fileID, path: path, origin: origin})
}
