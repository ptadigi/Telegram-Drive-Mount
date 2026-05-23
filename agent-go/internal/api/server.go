package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	agentauth "telegram-drive-agent/internal/auth"
	"telegram-drive-agent/internal/config"
)

type Server struct {
	startedAt time.Time
	version   string
	config    config.Config
	auth      *agentauth.Service
}

func NewServer(version string, cfg config.Config, authService *agentauth.Service) *Server {
	return &Server{
		startedAt: time.Now(),
		version:   version,
		config:    cfg,
		auth:      authService,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/info", s.handleInfo)
	mux.HandleFunc("GET /v1/config", s.handleConfig)
	mux.HandleFunc("GET /v1/database/status", s.handleDatabaseStatus)
	mux.HandleFunc("GET /v1/auth/status", s.handleAuthStatus)
	mux.HandleFunc("PUT /v1/auth/config", s.handleAuthConfig)
	mux.HandleFunc("POST /v1/auth/start", s.handleAuthStart)
	mux.HandleFunc("POST /v1/auth/code", s.handleAuthCode)
	mux.HandleFunc("POST /v1/auth/password", s.handleAuthPassword)
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
			"telegram_auth":    true,
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

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.auth.Status())
}

func (s *Server) handleAuthConfig(w http.ResponseWriter, r *http.Request) {
	var input struct {
		APIID   int    `json:"api_id"`
		APIHash string `json:"api_hash"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.APIID == 0 || input.APIHash == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("vui lòng nhập API ID và API Hash"))
		return
	}
	s.auth.UpdateTelegramConfig(input.APIID, input.APIHash)
	writeJSON(w, http.StatusOK, s.auth.Status())
}

func (s *Server) handleAuthStart(w http.ResponseWriter, r *http.Request) {
	var input agentauth.StartLoginInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.auth.StartLogin(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAuthCode(w http.ResponseWriter, r *http.Request) {
	var input agentauth.SubmitCodeInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.auth.SubmitCode(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAuthPassword(w http.ResponseWriter, r *http.Request) {
	var input agentauth.SubmitPasswordInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.auth.SubmitPassword(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
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

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"error": err.Error(),
	})
}

func errBadRequest(message string) error {
	return errors.New(message)
}
