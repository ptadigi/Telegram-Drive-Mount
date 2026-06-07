package drive

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

// chunkCache stores fetched Telegram byte ranges on disk so repeat reads of
// the same region (video scrubbing, re-open) avoid another MTProto round trip.
// Keyed by fileID+offset+length, value is the raw bytes. LRU eviction keeps
// total size under maxBytes. Files live under dataDir/chunks/<aa>/<hash>.
type chunkCache struct {
	dir      string
	maxBytes int64
	mu       sync.Mutex
	used     int64
	entries  map[string]*chunkMeta
}

type chunkMeta struct {
	path     string
	size     int64
	accessed int64
}

func newChunkCache(dataDir string, maxBytes int64) *chunkCache {
	if maxBytes <= 0 {
		maxBytes = 5 * 1024 * 1024 * 1024
	}
	c := &chunkCache{
		dir:      filepath.Join(dataDir, "chunks"),
		maxBytes: maxBytes,
		entries:  map[string]*chunkMeta{},
	}
	_ = os.MkdirAll(c.dir, 0o755)
	c.scan()
	return c
}

func chunkKey(fileID string, offset, length int64) string {
	sum := sha1.Sum([]byte(fileID + ":" + strconv.FormatInt(offset, 10) + ":" + strconv.FormatInt(length, 10)))
	return hex.EncodeToString(sum[:])
}

func (c *chunkCache) pathFor(key string) string {
	return filepath.Join(c.dir, key[:2], key)
}

func (c *chunkCache) scan() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = filepath.Walk(c.dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		key := filepath.Base(p)
		c.entries[key] = &chunkMeta{path: p, size: info.Size(), accessed: info.ModTime().UnixNano()}
		c.used += info.Size()
		return nil
	})
}

// Get returns cached bytes for the key, or nil if not present.
func (c *chunkCache) Get(key string) []byte {
	c.mu.Lock()
	meta, ok := c.entries[key]
	c.mu.Unlock()
	if !ok {
		return nil
	}
	data, err := os.ReadFile(meta.path)
	if err != nil {
		c.mu.Lock()
		delete(c.entries, key)
		c.used -= meta.size
		c.mu.Unlock()
		return nil
	}
	now := time.Now()
	_ = os.Chtimes(meta.path, now, now)
	c.mu.Lock()
	meta.accessed = now.UnixNano()
	c.mu.Unlock()
	return data
}

// Put stores bytes under key and evicts LRU entries past the budget.
func (c *chunkCache) Put(key string, data []byte) {
	if len(data) == 0 {
		return
	}
	p := c.pathFor(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return
	}
	c.mu.Lock()
	if old, ok := c.entries[key]; ok {
		c.used -= old.size
	}
	c.entries[key] = &chunkMeta{path: p, size: int64(len(data)), accessed: time.Now().UnixNano()}
	c.used += int64(len(data))
	c.evictLocked()
	c.mu.Unlock()
}

func (c *chunkCache) evictLocked() {
	if c.used <= c.maxBytes {
		return
	}
	metas := make([]*chunkMeta, 0, len(c.entries))
	keys := make([]string, 0, len(c.entries))
	for k, m := range c.entries {
		metas = append(metas, m)
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return c.entries[keys[i]].accessed < c.entries[keys[j]].accessed
	})
	for _, k := range keys {
		if c.used <= c.maxBytes {
			break
		}
		m := c.entries[k]
		if err := os.Remove(m.path); err == nil || os.IsNotExist(err) {
			c.used -= m.size
			delete(c.entries, k)
		}
	}
}