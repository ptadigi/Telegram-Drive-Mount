package drive

import (
	"context"
	"errors"
	"io"
	"sync"
)

// chunkCoalesce makes sure that two callers asking for the same Telegram
// stream range run only one MTProto fetch. The second caller blocks on the
// in-flight fetch and re-uses its bytes; this dramatically reduces FLOOD_WAIT
// risk and bandwidth when multiple devices stream the same file at once.
type chunkCoalesce struct {
	mu       sync.Mutex
	inflight map[string]*coalesceEntry
}

type coalesceEntry struct {
	wait chan struct{}
	data []byte
	err  error
	mime string
	size int64
}

func newChunkCoalesce() *chunkCoalesce {
	return &chunkCoalesce{inflight: map[string]*coalesceEntry{}}
}

// Do executes fn (which streams the chunk and writes into a private buffer).
// Concurrent callers with the same key share fn's output once it completes.
//
// fn must accept the writer it should write to. The caller's w receives the
// same bytes the leader produced.
func (c *chunkCoalesce) Do(ctx context.Context, key string, w io.Writer, fn func(out io.Writer) (StreamResult, error)) (StreamResult, error) {
	if key == "" {
		return fn(w)
	}
	c.mu.Lock()
	if existing, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		select {
		case <-existing.wait:
		case <-ctx.Done():
			return StreamResult{}, ctx.Err()
		}
		if existing.err != nil {
			return StreamResult{}, existing.err
		}
		if _, err := w.Write(existing.data); err != nil {
			return StreamResult{}, err
		}
		return StreamResult{Size: existing.size, MimeType: existing.mime}, nil
	}
	entry := &coalesceEntry{wait: make(chan struct{})}
	c.inflight[key] = entry
	c.mu.Unlock()

	buf := &chunkBuf{}
	res, err := fn(buf)
	c.mu.Lock()
	delete(c.inflight, key)
	c.mu.Unlock()
	entry.data = buf.bytes
	entry.err = err
	entry.size = res.Size
	entry.mime = res.MimeType
	close(entry.wait)
	if err != nil {
		return StreamResult{}, err
	}
	if _, err := w.Write(entry.data); err != nil {
		return StreamResult{}, err
	}
	return res, nil
}

type chunkBuf struct {
	bytes []byte
}

func (c *chunkBuf) Write(p []byte) (int, error) {
	c.bytes = append(c.bytes, p...)
	return len(p), nil
}

// ErrCoalesceCancelled is returned when an upstream context cancels mid-fetch.
var ErrCoalesceCancelled = errors.New("stream coalesce cancelled")