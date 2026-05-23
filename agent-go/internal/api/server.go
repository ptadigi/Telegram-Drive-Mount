package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type Server struct {
	startedAt time.Time
	version   string
}

func NewServer(version string) *Server {
	return &Server{
		startedAt: time.Now(),
		version:   version,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/info", s.handleInfo)
	return withJSON(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"service":   "telegram-drive-agent",
		"version":   s.version,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":       "Telegram Drive Agent",
		"version":    s.version,
		"started_at": s.startedAt.UTC().Format(time.RFC3339),
		"uptime_sec": int(time.Since(s.startedAt).Seconds()),
		"features": map[string]bool{
			"telegram_storage": false,
			"local_sync":       false,
			"media_streaming":  false,
			"webdav":           false,
		},
	})
}

func withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
