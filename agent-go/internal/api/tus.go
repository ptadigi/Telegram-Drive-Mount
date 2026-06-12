package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tushandler "github.com/tus/tusd/v2/pkg/handler"

	"github.com/tus/tusd/v2/pkg/filelocker"
	"github.com/tus/tusd/v2/pkg/filestore"

	"telegram-drive-agent/internal/drive"
)

// tusTempTTL is how long an incomplete tus temp upload may sit on disk before
// the janitor removes it. Interrupted/abandoned uploads (browser closed, failed
// chunk, parallel-upload leftovers) otherwise accumulate as 0-byte temp + .info
// files. Active uploads refresh their mtime as chunks arrive, so a generous TTL
// never deletes an in-progress upload.
const tusTempTTL = 12 * time.Hour


// newTusHandler builds an embedded tus resumable-upload handler mounted under
// basePath. Completed uploads are imported into the drive pipeline (DB + queue
// + Telegram sync) via the OnUploadFinish hook, then the temp tus files are
// removed. Chunked uploads sidestep reverse-proxy body-size limits.
func (s *Server) newTusHandler(basePath string) (http.Handler, error) {
	tusDir := filepath.Join(s.config.DataDir, "uploads", "tus")
	if err := os.MkdirAll(tusDir, 0o755); err != nil {
		return nil, err
	}
	store := filestore.New(tusDir)
	locker := filelocker.New(tusDir)
	composer := tushandler.NewStoreComposer()
	store.UseIn(composer)
	locker.UseIn(composer)

	handler, err := tushandler.NewHandler(tushandler.Config{
		BasePath:              basePath,
		StoreComposer:         composer,
		DisableDownload:       true,
		NotifyCompleteUploads: true,
		// Behind a reverse proxy the request scheme is http; trust
		// X-Forwarded-Proto/Host so the returned upload URL is https and the
		// client can PATCH chunks without mixed-content blocking.
		RespectForwardedHeaders: true,
		MaxSize:                 0,
	})
	if err != nil {
		return nil, err
	}

	// Import finished uploads into the drive pipeline.
	go func() {
		for event := range handler.CompleteUploads {
			s.importTusUpload(event)
		}
	}()

	// Janitor: periodically sweep abandoned tus temp files so a self-hosted
	// instance never accumulates dead 0-byte uploads from interrupted sessions.
	go s.runTusJanitor(tusDir)

	return http.StripPrefix(basePath, handler), nil
}

// runTusJanitor removes tus temp files older than tusTempTTL. It runs hourly
// and is safe against in-progress uploads because tusd touches the data + .info
// files as chunks arrive, keeping their mtime fresh well within the TTL.
func (s *Server) runTusJanitor(tusDir string) {
	sweep := func() {
		entries, err := os.ReadDir(tusDir)
		if err != nil {
			return
		}
		cutoff := time.Now().Add(-tusTempTTL)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				_ = os.Remove(filepath.Join(tusDir, entry.Name()))
			}
		}
	}
	sweep()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		sweep()
	}
}


func (s *Server) importTusUpload(event tushandler.HookEvent) {
	info := event.Upload
	storedPath := info.Storage["Path"]
	if storedPath == "" {
		return
	}
	filename := info.MetaData["filename"]
	if filename == "" {
		filename = "upload.bin"
	}
	folderID := info.MetaData["folder_id"]
	relativePath := info.MetaData["relative_path"]
	mimeHint := info.MetaData["filetype"]

	ctx := context.Background()
	if uid := info.MetaData["user_id"]; uid != "" {
		ctx = drive.WithUser(ctx, uid)
	}

	// Open the assembled tus file and import it through the normal upload path
	// (copies into uploads/ with proper id naming, registers DB + transfer +
	// Telegram queue). Then remove the tus temp + info files.
	f, err := os.Open(storedPath)
	if err != nil {
		s.drive.LogSyncError("tus_import_open_failed", filename, info.Size, err)
		return
	}
	_, err = s.drive.SaveStreamFile(ctx, f, filename, mimeHint, folderID, relativePath)
	_ = f.Close()
	if err != nil {
		s.drive.LogSyncError("tus_import_failed", filename, info.Size, err)
		return
	}
	_ = os.Remove(storedPath)
	_ = os.Remove(storedPath + ".info")
}
