package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"telegram-drive-agent/internal/config"
)

// TestAuthGateBlocksByDefault ensures every /v1/* endpoint requires a session
// when no Basic Auth is configured (open mode).
func TestAuthGateBlocksByDefault(t *testing.T) {
	srv := NewServer("test", config.Config{Auth: config.AuthConfig{Mode: "open"}}, nil, nil, nil, nil, nil, nil)
	handler := srv.Handler()
	guarded := []string{
		"/v1/mount",
		"/v1/transfers",
		"/v1/files",
		"/v1/audit",
		"/v1/storage",
		"/v1/auth/api-config",
		"/v1/cache",
		"/v1/sync/roots",
		"/v1/drive/contents",
		"/webdav/",
	}
	for _, path := range guarded {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for %s, got %d (body=%q)", path, w.Code, strings.TrimSpace(w.Body.String()))
		}
	}
}

// TestPublicPathsBypassAuth ensures /health and /.td-check are accessible
// without a session even when no auth/user service is configured.
func TestPublicPathsBypassAuth(t *testing.T) {
	srv := NewServer("test", config.Config{Auth: config.AuthConfig{Mode: "open"}}, nil, nil, nil, nil, nil, nil)
	handler := srv.Handler()
	allowed := []string{
		"/health",
		"/.td-check",
	}
	for _, path := range allowed {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusUnauthorized {
			t.Errorf("expected non-401 for public %s, got %d", path, w.Code)
		}
	}
}

// TestBasicAuthChallenged verifies that when Basic Auth is configured,
// missing credentials yield 401 with WWW-Authenticate.
func TestBasicAuthChallenged(t *testing.T) {
	srv := NewServer("test", config.Config{Auth: config.AuthConfig{Mode: "basic", Username: "admin", Password: "secret"}}, nil, nil, nil, nil, nil, nil)
	handler := srv.Handler()
	req := httptest.NewRequest(http.MethodGet, "/v1/mount", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without basic auth, got %d", w.Code)
	}
	if challenge := w.Header().Get("WWW-Authenticate"); !strings.HasPrefix(challenge, "Basic") {
		t.Fatalf("expected WWW-Authenticate Basic header, got %q", challenge)
	}
}

