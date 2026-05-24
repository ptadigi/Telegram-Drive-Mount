package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	agentauth "telegram-drive-agent/internal/auth"
	"telegram-drive-agent/internal/config"
	"telegram-drive-agent/internal/drive"
	"telegram-drive-agent/internal/tunnel"
)

type Server struct {
	startedAt time.Time
	version   string
	config    config.Config
	auth      *agentauth.Service
	drive     *drive.Service
	tunnel    *tunnel.Service
	shareRate *rateLimiter
	authMu    sync.RWMutex
	authCfg   config.AuthConfig
}

func NewServer(version string, cfg config.Config, authService *agentauth.Service, driveService *drive.Service, tunnelService *tunnel.Service) *Server {
	return &Server{
		startedAt: time.Now(),
		version:   version,
		config:    cfg,
		auth:      authService,
		drive:     driveService,
		tunnel:    tunnelService,
		shareRate: newRateLimiter(20, time.Minute),
		authCfg:   cfg.Auth,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/info", s.handleInfo)
	mux.HandleFunc("GET /v1/config", s.handleConfig)
	mux.HandleFunc("GET /v1/database/status", s.handleDatabaseStatus)
	mux.HandleFunc("GET /v1/transfers", s.handleTransfers)
	mux.HandleFunc("GET /v1/events", s.handleEvents)
	mux.HandleFunc("GET /v1/sync/roots", s.handleListSyncRoots)
	mux.HandleFunc("POST /v1/sync/roots", s.handleCreateSyncRoot)
	mux.HandleFunc("PUT /v1/sync/roots", s.handleUpdateSyncRoot)
	mux.HandleFunc("DELETE /v1/sync/roots", s.handleDeleteSyncRoot)
	mux.HandleFunc("POST /v1/sync/roots/scan", s.handleScanSyncRoot)
	mux.HandleFunc("GET /v1/files", s.handleListFiles)
	mux.HandleFunc("GET /v1/drive/contents", s.handleDriveContents)
	mux.HandleFunc("POST /v1/folders", s.handleCreateFolder)
	mux.HandleFunc("PUT /v1/folders/rename", s.handleRenameFolder)
	mux.HandleFunc("POST /v1/folders/trash", s.handleTrashFolder)
	mux.HandleFunc("POST /v1/folders/restore", s.handleRestoreFolder)
	mux.HandleFunc("PUT /v1/files/rename", s.handleRenameFile)
	mux.HandleFunc("POST /v1/files/trash", s.handleTrashFile)
	mux.HandleFunc("POST /v1/files/restore", s.handleRestoreFile)
	mux.HandleFunc("GET /v1/trash", s.handleListTrash)
	mux.HandleFunc("GET /v1/search", s.handleSearch)
	mux.HandleFunc("GET /v1/starred", s.handleListStarred)
	mux.HandleFunc("PUT /v1/files/move", s.handleMoveFile)
	mux.HandleFunc("PUT /v1/files/star", s.handleStarFile)
	mux.HandleFunc("DELETE /v1/files", s.handleDeleteFile)
	mux.HandleFunc("PUT /v1/folders/move", s.handleMoveFolder)
	mux.HandleFunc("PUT /v1/folders/star", s.handleStarFolder)
	mux.HandleFunc("DELETE /v1/folders", s.handleDeleteFolder)
	mux.HandleFunc("GET /v1/folders/zip", s.handleZipFolder)
	mux.HandleFunc("POST /v1/bundle/zip", s.handleZipBundle)
	mux.HandleFunc("GET /v1/shares", s.handleListShares)
	mux.HandleFunc("POST /v1/shares", s.handleCreateShare)
	mux.HandleFunc("PUT /v1/shares", s.handleUpdateShare)
	mux.HandleFunc("DELETE /v1/shares", s.handleDeleteShare)
	mux.HandleFunc("GET /v1/share/config", s.handleShareConfigGet)
	mux.HandleFunc("PUT /v1/share/config", s.handleShareConfigPut)
	mux.HandleFunc("GET /v1/storage", s.handleStorageGet)
	mux.HandleFunc("PUT /v1/storage", s.handleStoragePut)
	mux.HandleFunc("POST /v1/storage/channel", s.handleCreateStorageChannel)
	mux.HandleFunc("GET /v1/audit", s.handleListAudit)
	mux.HandleFunc("GET /v1/auth/api-config", s.handleAPIAuthGet)
	mux.HandleFunc("PUT /v1/auth/api-config", s.handleAPIAuthPut)
	mux.Handle("/webdav/", s.webdavHandler())
	mux.Handle("/webdav", s.webdavHandler())
	mux.HandleFunc("GET /v1/cache", s.handleCacheStats)
	mux.HandleFunc("PUT /v1/cache", s.handleCacheConfig)
	mux.HandleFunc("POST /v1/cache/cleanup", s.handleCacheCleanup)
	mux.HandleFunc("POST /v1/share/tunnel", s.handleShareTunnel)
	mux.HandleFunc("GET /.td-check", s.handleShareHealthCheck)
	mux.HandleFunc("GET /share/{slug}", s.handleSharePage)
	mux.HandleFunc("POST /share/{slug}/unlock", s.handleShareUnlock)
	mux.HandleFunc("GET /share/{slug}/raw", s.handleShareRaw)
	mux.HandleFunc("GET /v1/files/download", s.handleDownloadFile)
	mux.HandleFunc("GET /v1/files/stream", s.handleStreamFile)
	mux.HandleFunc("GET /v1/files/thumbnail", s.handleFileThumbnail)
	mux.HandleFunc("POST /v1/files/upload", s.handleUploadFile)
	mux.HandleFunc("POST /v1/files/sync", s.handleSyncFiles)
	mux.HandleFunc("POST /v1/files/demo", s.handleSeedDemoFile)
	mux.HandleFunc("GET /v1/auth/status", s.handleAuthStatus)
	mux.HandleFunc("PUT /v1/auth/config", s.handleAuthConfig)
	mux.HandleFunc("POST /v1/auth/reset", s.handleAuthReset)
	mux.HandleFunc("POST /v1/auth/start", s.handleAuthStart)
	mux.HandleFunc("POST /v1/auth/code", s.handleAuthCode)
	mux.HandleFunc("POST /v1/auth/password", s.handleAuthPassword)
	return withJSON(s.withAuth(withCORS(mux)))
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.authMu.RLock()
		cfg := s.authCfg
		s.authMu.RUnlock()
		if cfg.Mode != "basic" || cfg.Password == "" {
			next.ServeHTTP(w, r)
			return
		}
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		user := cfg.Username
		if user == "" {
			user = "admin"
		}
		gotUser, gotPass, ok := r.BasicAuth()
		if !ok || gotUser != user || gotPass != cfg.Password {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"Ổ Đĩa Cloud Ảo\"")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) UpdateAuthConfig(cfg config.AuthConfig) {
	s.authMu.Lock()
	s.authCfg = cfg
	s.authMu.Unlock()
}

func (s *Server) AuthConfig() config.AuthConfig {
	s.authMu.RLock()
	defer s.authMu.RUnlock()
	return s.authCfg
}

func isPublicPath(path string) bool {
	if path == "/health" || path == "/.td-check" {
		return true
	}
	if strings.HasPrefix(path, "/share/") {
		return true
	}
	return false
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

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errBadRequest("trình duyệt không hỗ trợ realtime stream"))
		return
	}
	events := s.drive.Events().Subscribe(r.Context())
	for event := range events {
		_, _ = w.Write([]byte("event: " + event.Type + "\n"))
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(event.JSON())
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
	}
}

func (s *Server) handleTransfers(w http.ResponseWriter, r *http.Request) {
	transfers, err := s.drive.ListTransfers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transfers": transfers})
}

func (s *Server) handleListSyncRoots(w http.ResponseWriter, r *http.Request) {
	roots, err := s.drive.ListSyncRoots(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roots": roots})
}

func (s *Server) handleCreateSyncRoot(w http.ResponseWriter, r *http.Request) {
	var input drive.CreateSyncRootInput
	if !decodeJSON(w, r, &input) {
		return
	}
	root, err := s.drive.CreateSyncRoot(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	roots, _ := s.drive.ListSyncRoots(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"root": root, "roots": roots})
}

func (s *Server) handleUpdateSyncRoot(w http.ResponseWriter, r *http.Request) {
	var input drive.UpdateSyncRootInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.drive.UpdateSyncRoot(r.Context(), input); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	roots, _ := s.drive.ListSyncRoots(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"roots": roots})
}

func (s *Server) handleDeleteSyncRoot(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("thiếu id thư mục đồng bộ"))
		return
	}
	if err := s.drive.DeleteSyncRoot(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	roots, _ := s.drive.ListSyncRoots(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"roots": roots})
}

func (s *Server) handleScanSyncRoot(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID string `json:"id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.ID == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("thiếu id thư mục đồng bộ"))
		return
	}
	if err := s.drive.ScanSyncRoot(r.Context(), input.ID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	roots, _ := s.drive.ListSyncRoots(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"roots": roots})
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	files, err := s.drive.ListFiles(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (s *Server) handleDriveContents(w http.ResponseWriter, r *http.Request) {
	folderID := r.URL.Query().Get("folder_id")
	contents, err := s.drive.ListFolderContents(r.Context(), folderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, contents)
}

func (s *Server) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	var input drive.CreateFolderInput
	if !decodeJSON(w, r, &input) {
		return
	}
	folder, err := s.drive.CreateFolder(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	contents, err := s.drive.ListFolderContents(r.Context(), input.ParentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folder": folder, "contents": contents})
}

func (s *Server) handleRenameFolder(w http.ResponseWriter, r *http.Request) {
	var input drive.RenameInput
	if !decodeJSON(w, r, &input) {
		return
	}
	folder, err := s.drive.RenameFolder(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folder": folder})
}

func (s *Server) handleRenameFile(w http.ResponseWriter, r *http.Request) {
	var input drive.RenameInput
	if !decodeJSON(w, r, &input) {
		return
	}
	file, err := s.drive.RenameFile(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"file": file})
}

func (s *Server) handleTrashFile(w http.ResponseWriter, r *http.Request) {
	var input drive.IDInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.drive.TrashFile(r.Context(), input.ID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleTrashFolder(w http.ResponseWriter, r *http.Request) {
	var input drive.IDInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.drive.TrashFolder(r.Context(), input.ID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRestoreFile(w http.ResponseWriter, r *http.Request) {
	var input drive.IDInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.drive.RestoreFile(r.Context(), input.ID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRestoreFolder(w http.ResponseWriter, r *http.Request) {
	var input drive.IDInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.drive.RestoreFolder(r.Context(), input.ID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	result, err := s.drive.Search(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleListStarred(w http.ResponseWriter, r *http.Request) {
	contents, err := s.drive.ListStarred(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, contents)
}

func (s *Server) handleMoveFile(w http.ResponseWriter, r *http.Request) {
	var input drive.MoveInput
	if !decodeJSON(w, r, &input) {
		return
	}
	file, err := s.drive.MoveFile(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"file": file})
}

func (s *Server) handleMoveFolder(w http.ResponseWriter, r *http.Request) {
	var input drive.MoveInput
	if !decodeJSON(w, r, &input) {
		return
	}
	folder, err := s.drive.MoveFolder(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folder": folder})
}

func (s *Server) handleStarFile(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID      string `json:"id"`
		Starred bool   `json:"starred"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.drive.StarFile(r.Context(), input.ID, input.Starred); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleStarFolder(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID      string `json:"id"`
		Starred bool   `json:"starred"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := s.drive.StarFolder(r.Context(), input.ID, input.Starred); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := s.drive.PermanentDeleteFile(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := s.drive.PermanentDeleteFolder(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleZipBundle(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FileIDs   []string `json:"file_ids"`
		FolderIDs []string `json:"folder_ids"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if len(input.FileIDs) == 0 && len(input.FolderIDs) == 0 {
		writeError(w, http.StatusBadRequest, errBadRequest("vui lòng chọn ít nhất một mục"))
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''bundle.zip")
	if err := s.drive.ZipBundle(r.Context(), input.FileIDs, input.FolderIDs, w); err != nil {
		writeError(w, http.StatusInternalServerError, err)
	}
}

func (s *Server) handleZipFolder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("thiếu id thư mục"))
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=folder.zip")
	if err := s.drive.ZipFolder(r.Context(), id, w); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
}

func (s *Server) handleListTrash(w http.ResponseWriter, r *http.Request) {
	contents, err := s.drive.ListTrash(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, contents)
}

func (s *Server) handleListShares(w http.ResponseWriter, r *http.Request) {
	shares, err := s.drive.ListShares(r.Context(), r.URL.Query().Get("target_kind"), r.URL.Query().Get("target_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shares": shares})
}

func (s *Server) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	var input drive.CreateShareInput
	if !decodeJSON(w, r, &input) {
		return
	}
	share, err := s.drive.CreateShare(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"share": share})
}

func (s *Server) handleUpdateShare(w http.ResponseWriter, r *http.Request) {
	var input drive.UpdateShareInput
	if !decodeJSON(w, r, &input) {
		return
	}
	share, err := s.drive.UpdateShare(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"share": share})
}

func (s *Server) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("thiếu id share"))
		return
	}
	if err := s.drive.DeleteShare(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.drive.CacheStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cache": stats})
}

func (s *Server) handleCacheConfig(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Mode     string `json:"mode"`
		MaxBytes int64  `json:"max_bytes"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Mode != "smart" && input.Mode != "cloud_only" && input.Mode != "mirror" {
		writeError(w, http.StatusBadRequest, errBadRequest("chế độ cache không hợp lệ"))
		return
	}
	s.drive.SetCachePolicy(input.Mode, input.MaxBytes)
	stats, _ := s.drive.CacheStats(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"cache": stats})
}

func (s *Server) handleCacheCleanup(w http.ResponseWriter, r *http.Request) {
	removed, err := s.drive.CleanupCache(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	stats, _ := s.drive.CacheStats(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"removed": removed, "cache": stats})
}

func (s *Server) handleAPIAuthGet(w http.ResponseWriter, r *http.Request) {
	cfg := s.AuthConfig()
	writeJSON(w, http.StatusOK, map[string]any{"auth": map[string]any{
		"mode":         cfg.Mode,
		"username":     cfg.Username,
		"has_password": cfg.Password != "",
	}})
}

func (s *Server) handleAPIAuthPut(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Mode     string `json:"mode"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode != "open" && mode != "basic" {
		writeError(w, http.StatusBadRequest, errBadRequest("chế độ auth không hợp lệ"))
		return
	}
	current := s.AuthConfig()
	cfg := config.AuthConfig{Mode: mode, Username: strings.TrimSpace(input.Username), Password: current.Password}
	if input.Password != "" {
		cfg.Password = input.Password
	}
	if mode == "basic" {
		if cfg.Username == "" {
			cfg.Username = "admin"
		}
		if cfg.Password == "" {
			writeError(w, http.StatusBadRequest, errBadRequest("vui lòng nhập mật khẩu"))
			return
		}
	}
	s.UpdateAuthConfig(cfg)
	writeJSON(w, http.StatusOK, map[string]any{"auth": map[string]any{"mode": cfg.Mode, "username": cfg.Username, "has_password": cfg.Password != ""}})
}

func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			limit = parsed
		}
	}
	entries, err := s.drive.ListAudit(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (s *Server) handleCreateStorageChannel(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title string `json:"title"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	settings, err := s.drive.CreateStorageChannel(r.Context(), input.Title)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"storage": settings})
}

func (s *Server) handleStorageGet(w http.ResponseWriter, r *http.Request) {
	settings, err := s.drive.GetStorageSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"storage": settings})
}

func (s *Server) handleStoragePut(w http.ResponseWriter, r *http.Request) {
	var input drive.UpdateStorageInput
	if !decodeJSON(w, r, &input) {
		return
	}
	settings, err := s.drive.UpdateStorageSettings(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.drive.WriteAudit(r.Context(), "settings", "storage.update", "storage", settings.PeerKind, map[string]any{"channel_id": settings.ChannelID, "title": settings.Title})
	writeJSON(w, http.StatusOK, map[string]any{"storage": settings})
}

func (s *Server) handleShareConfigGet(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.drive.GetShareConfig(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"config": cfg})
}

func (s *Server) handleShareConfigPut(w http.ResponseWriter, r *http.Request) {
	var input drive.UpdateShareConfigInput
	if !decodeJSON(w, r, &input) {
		return
	}
	host := r.Host
	if host == "" {
		host = s.config.Addr()
	}
	localBase := "http://" + host
	cfg, err := s.drive.UpdateShareConfig(r.Context(), input, localBase)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"config": cfg})
}

func (s *Server) handleShareTunnel(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Action string `json:"action"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	switch input.Action {
	case "start":
		status, err := s.tunnel.Start(s.config.Port)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tunnel": status})
	case "stop":
		s.tunnel.Stop()
		writeJSON(w, http.StatusOK, map[string]any{"tunnel": s.tunnel.Status()})
	default:
		writeError(w, http.StatusBadRequest, errBadRequest("action không hợp lệ"))
	}
}

func (s *Server) handleShareHealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "telegram-drive-share"})
}

func (s *Server) handleSharePage(w http.ResponseWriter, r *http.Request) {
	if !s.shareRate.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, errBadRequest("quá nhiều yêu cầu, vui lòng thử lại sau"))
		return
	}
	slug := r.PathValue("slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("thiếu slug"))
		return
	}
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(renderSharePageHTML(slug)))
		return
	}
	resolved, err := s.drive.ResolveShare(r.Context(), slug, r.URL.Query().Get("password"))
	if err != nil {
		if drive.IsSharePasswordRequired(err) {
			writeJSON(w, http.StatusOK, map[string]any{"requires_password": true, "share": map[string]any{"slug": slug, "has_password": true, "target_kind": resolved.Share.TargetKind}})
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	share := resolved.Share
	payload := map[string]any{
		"share": map[string]any{
			"slug":          share.Slug,
			"target_kind":   share.TargetKind,
			"has_password":  share.HasPassword,
			"expires_at":    share.ExpiresAt,
			"max_downloads": share.MaxDownloads,
			"access_count":  share.AccessCount,
		},
	}
	if resolved.File != nil {
		file := *resolved.File
		payload["file"] = map[string]any{
			"name":       file.Name,
			"size":       file.Size,
			"mime_type":  file.MimeType,
			"kind":       file.Kind,
			"updated_at": file.UpdatedAt,
		}
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleShareUnlock(w http.ResponseWriter, r *http.Request) {
	if !s.shareRate.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, errBadRequest("quá nhiều yêu cầu, vui lòng thử lại sau"))
		return
	}
	slug := r.PathValue("slug")
	var input struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	resolved, err := s.drive.ResolveShare(r.Context(), slug, input.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"slug": resolved.Share.Slug, "ok": true})
}

func (s *Server) handleShareRaw(w http.ResponseWriter, r *http.Request) {
	if !s.shareRate.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, errBadRequest("quá nhiều yêu cầu, vui lòng thử lại sau"))
		return
	}
	slug := r.PathValue("slug")
	password := r.URL.Query().Get("password")
	resolved, err := s.drive.ResolveShare(r.Context(), slug, password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	if resolved.Share.TargetKind == "folder" {
		s.drive.RecordShareAccess(r.Context(), resolved.Share.ID)
		if err := s.drive.StreamFolderShareZip(r.Context(), resolved.Share, w); err != nil {
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	file, err := s.drive.DownloadableForShare(r.Context(), resolved.Share)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	s.drive.RecordShareAccess(r.Context(), resolved.Share.ID)
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+urlQueryEscape(file.Name))
	http.ServeFile(w, r, file.LocalPath)
}

func (s *Server) handleStreamFile(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("thiếu id file"))
		return
	}
	stream, err := s.drive.GetStreamableFile(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if stream.Source == drive.StreamFromCache {
		w.Header().Set("Content-Type", stream.MimeType)
		http.ServeFile(w, r, stream.LocalPath)
		return
	}
	offset, length, err := parseRange(r.Header.Get("Range"), stream.Size)
	if err != nil {
		writeError(w, http.StatusRequestedRangeNotSatisfiable, err)
		return
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", stream.MimeType)
	if length > 0 && length < stream.Size {
		w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(offset, 10)+"-"+strconv.FormatInt(offset+length-1, 10)+"/"+strconv.FormatInt(stream.Size, 10))
		w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.Header().Set("Content-Length", strconv.FormatInt(stream.Size, 10))
		w.WriteHeader(http.StatusOK)
	}
	if _, err := s.drive.StreamFromTelegram(r.Context(), id, offset, length, w); err != nil {
		// Stream đã bắt đầu, không thể trả status code khác. Ghi log thôi.
		return
	}
}

func parseRange(header string, total int64) (int64, int64, error) {
	if header == "" {
		return 0, total, nil
	}
	if !strings.HasPrefix(header, "bytes=") {
		return 0, 0, errBadRequest("range không hợp lệ")
	}
	spec := strings.TrimPrefix(header, "bytes=")
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, errBadRequest("range không hợp lệ")
	}
	var start, end int64
	if parts[0] != "" {
		if _, err := fmt.Sscanf(parts[0], "%d", &start); err != nil {
			return 0, 0, errBadRequest("range không hợp lệ")
		}
	}
	if parts[1] != "" {
		if _, err := fmt.Sscanf(parts[1], "%d", &end); err != nil {
			return 0, 0, errBadRequest("range không hợp lệ")
		}
	} else {
		end = total - 1
	}
	if total > 0 && end >= total {
		end = total - 1
	}
	if start > end {
		return 0, 0, errBadRequest("range vượt quá kích thước")
	}
	return start, end - start + 1, nil
}

func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("thiếu id file"))
		return
	}

	file, err := s.drive.GetDownloadableFile(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+urlQueryEscape(file.Name))
	http.ServeFile(w, r, file.LocalPath)
}

func (s *Server) handleFileThumbnail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errBadRequest("thiếu id file"))
		return
	}
	thumb, err := s.drive.GetThumbnail(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.Header().Set("Content-Type", thumb.MimeType)
	http.ServeFile(w, r, thumb.Path)
}

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	_, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	file, err := s.drive.SaveUploadedFile(r.Context(), header, r.FormValue("folder_id"), r.FormValue("relative_path"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"file": file})
}

func (s *Server) handleSyncFiles(w http.ResponseWriter, r *http.Request) {
	result, err := s.drive.SyncPendingToTelegram(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	contents, err := s.drive.ListFolderContents(r.Context(), "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sync": result, "contents": contents, "files": contents.Files})
}

func (s *Server) handleSeedDemoFile(w http.ResponseWriter, r *http.Request) {
	if err := s.drive.SeedDemoFile(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	contents, err := s.drive.ListFolderContents(r.Context(), "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contents": contents, "files": contents.Files})
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.auth.Status(r.Context()))
}

func (s *Server) handleAuthReset(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.ResetLogin(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.auth.Status(r.Context()))
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
	writeJSON(w, http.StatusOK, s.auth.Status(r.Context()))
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
