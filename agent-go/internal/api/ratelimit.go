package api

import (
	"net/http"
	"sync"
	"time"
)

type rateBucket struct {
	count int
	reset time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{buckets: map[string]*rateBucket{}, limit: limit, window: window}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	bucket, ok := l.buckets[key]
	if !ok || now.After(bucket.reset) {
		l.buckets[key] = &rateBucket{count: 1, reset: now.Add(l.window)}
		return true
	}
	if bucket.count >= l.limit {
		return false
	}
	bucket.count++
	return true
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.RemoteAddr; ip != "" {
		return ip
	}
	return "unknown"
}
