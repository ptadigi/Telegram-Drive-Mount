package drive

import (
	"bytes"
	"testing"
)

func TestChunkCachePutGet(t *testing.T) {
	c := newChunkCache(t.TempDir(), 1024*1024)
	key := chunkKey("file-1", 0, 1024)
	data := bytes.Repeat([]byte("x"), 512)
	c.Put(key, data)
	got := c.Get(key)
	if !bytes.Equal(got, data) {
		t.Fatalf("get mismatch: %d vs %d", len(got), len(data))
	}
}

func TestChunkCacheMiss(t *testing.T) {
	c := newChunkCache(t.TempDir(), 1024*1024)
	if got := c.Get(chunkKey("nope", 0, 1)); got != nil {
		t.Fatalf("expected nil on miss, got %d bytes", len(got))
	}
}

func TestChunkCacheLRUEvict(t *testing.T) {
	// budget fits ~3 chunks of 1000 bytes
	c := newChunkCache(t.TempDir(), 3000)
	blocks := map[string][]byte{}
	for i := 0; i < 6; i++ {
		key := chunkKey("file", int64(i), 1000)
		data := bytes.Repeat([]byte{byte('a' + i)}, 1000)
		blocks[key] = data
		c.Put(key, data)
	}
	if c.used > c.maxBytes {
		t.Fatalf("used %d exceeds budget %d", c.used, c.maxBytes)
	}
	// at least the last write must survive
	last := chunkKey("file", 5, 1000)
	if got := c.Get(last); got == nil {
		t.Fatalf("most recent chunk evicted")
	}
}

func TestChunkKeyStable(t *testing.T) {
	a := chunkKey("f", 10, 20)
	b := chunkKey("f", 10, 20)
	if a != b {
		t.Fatalf("chunkKey not deterministic")
	}
	if a == chunkKey("f", 10, 21) {
		t.Fatalf("chunkKey collision on different length")
	}
}