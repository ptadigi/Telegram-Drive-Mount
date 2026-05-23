package api

import (
	"encoding/json"
	"net/http"
	"time"

	"telegram-drive-agent/internal/config"
)

type Server struct {
	startedAt time.Time
	version   string
	config    config.Config
}

func NewServer(version string, cfg config.Config) *Server {
	return &Server{
		startedAt: time.Now(),
		version:   version,
		config:    cfg,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/info", s.handleInfo)
	mux.HandleFunc("GET /v1/config", s.handleConfig)
	mux.HandleFunc("GET /v1/database/status", s.handleDatabaseStatus)
	return withJSON(withCORS(mux))
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

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"host":          s.config.Host,
		"port":          s.config.Port,
		"data_dir":      s.config.DataDir,
		"database_path": s.config.DatabasePath,
		"telegram": map[string]any{
			"api_id_set":     s.config.Telegram.APIID != 0,
			"api_hash_set":   s.config.Telegram.APIHash != "",
			"session_path":   s.config.Telegram.SessionPath,
			"session_exists": fileExists(s.config.Telegram.SessionPath),
		},
	})
}

func (s *Server) handleDatabaseStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"path":   s.config.DatabasePath,
		"exists": fileExists(s.config.DatabasePath),
	})
}

func withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
