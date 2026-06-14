package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateBucket struct {
	count int
	reset time.Time
}

type rateLimiter struct {
	mu          sync.Mutex
	buckets     map[string]*rateBucket
	limit       int
	window      time.Duration
	lastCleanup time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{buckets: map[string]*rateBucket{}, limit: limit, window: window, lastCleanup: time.Now()}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	// Opportunistically evict expired buckets so the map can't grow without
	// bound on a public, internet-facing endpoint (one entry per distinct
	// client key would otherwise live forever). Sweeping at most once per
	// window keeps this O(n) work rare.
	if now.Sub(l.lastCleanup) >= l.window {
		for k, b := range l.buckets {
			if now.After(b.reset) {
				delete(l.buckets, k)
			}
		}
		l.lastCleanup = now
	}
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

// clientIP returns the request's source IP. X-Forwarded-For is only honored
// when the immediate peer is loopback (development setup behind localhost
// reverse proxy); otherwise it is ignored to prevent rate-limit bypass via
// arbitrary header injection.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if isTrustedProxyIP(host) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if host != "" {
		return host
	}
	return "unknown"
}

func isTrustedProxyIP(host string) bool {
	if host == "" {
		return false
	}
	parsed := net.ParseIP(host)
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback()
}
