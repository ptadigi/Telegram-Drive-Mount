package drive

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// telegramFolderName is the drive folder where files uploaded directly from a
// native Telegram client are imported, so they never get mixed into the user's
// own folder tree.
const telegramFolderName = "Telegram Folder"

// scanStateKey is the storage_settings-adjacent marker for the last channel
// message we have imported, kept per user so incremental scans stay cheap.
// We store it in a tiny key/value table created on demand.

// ensureScanStateTable creates the scan-state table once.
func (s *Service) ensureScanStateTable(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS channel_scan_state (
			user_id TEXT PRIMARY KEY,
			last_message_id INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`)
	return err
}

func (s *Service) lastScannedMessageID(ctx context.Context, userID string) (int, error) {
	if err := s.ensureScanStateTable(ctx); err != nil {
		return 0, err
	}
	var last int
	err := s.db.QueryRowContext(ctx, `SELECT last_message_id FROM channel_scan_state WHERE user_id = ?`, userID).Scan(&last)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return last, nil
}

func (s *Service) setLastScannedMessageID(ctx context.Context, userID string, id int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO channel_scan_state (user_id, last_message_id, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET last_message_id = excluded.last_message_id, updated_at = excluded.updated_at
	`, userID, id, time.Now().Unix())
	return err
}

// ensureTelegramFolder returns the id of the user's "Telegram Folder", creating
// it at the drive root if missing. Always present so the user sees it.
func (s *Service) ensureTelegramFolder(ctx context.Context) (string, error) {
	folder, err := s.getOrCreateFolder(ctx, "", telegramFolderName)
	if err != nil {
		return "", err
	}
	return folder.ID, nil
}

// EnsureTelegramFolder is the exported entrypoint so the scan worker can make
// the folder appear even before the first import.
func (s *Service) EnsureTelegramFolder(ctx context.Context) error {
	_, err := s.ensureTelegramFolder(ctx)
	return err
}

// ImportChannelFiles scans the storage channel for media not yet known to the
// drive and imports each as a metadata-only file (no download) into the
// "Telegram Folder". Idempotent: messages already present (by
// telegram_message_id) are skipped, and a per-user cursor avoids re-reading old
// history. Returns the number of files imported.
func (s *Service) ImportChannelFiles(ctx context.Context, scanner ChannelScanner) (int, error) {
	settings, err := s.GetStorageSettings(ctx)
	if err != nil {
		return 0, err
	}
	if settings.PeerKind != "channel" || settings.ChannelID == 0 {
		return 0, nil // nothing to scan until a storage channel is configured
	}
	userID := UserFromContext(ctx)

	// Make the folder visible even when there is nothing to import yet.
	folderID, err := s.ensureTelegramFolder(ctx)
	if err != nil {
		return 0, err
	}

	peer := StoragePeer{Kind: settings.PeerKind, ChannelID: settings.ChannelID, AccessHash: settings.AccessHash}
	cursor, err := s.lastScannedMessageID(ctx, userID)
	if err != nil {
		return 0, err
	}

	imported := 0
	maxLoops := 50 // safety cap: up to 50 * 100 = 5000 messages per run
	for loop := 0; loop < maxLoops; loop++ {
		batch, err := scanner.ScanChannelHistory(ctx, peer, cursor, 100)
		if err != nil {
			return imported, err
		}
		if len(batch) == 0 {
			break
		}
		for _, cf := range batch {
			if cf.MessageID > cursor {
				cursor = cf.MessageID
			}
			ok, ierr := s.importOneChannelFile(ctx, folderID, peer, cf)
			if ierr != nil {
				// Skip a bad message but keep going.
				s.LogSyncError("channel_import_failed", cf.Name, cf.Size, ierr)
				continue
			}
			if ok {
				imported++
			}
		}
		// Persist progress after each batch so a crash doesn't redo work.
		if err := s.setLastScannedMessageID(ctx, userID, cursor); err != nil {
			return imported, err
		}
		// Gentle pacing to avoid hammering the Telegram API.
		select {
		case <-ctx.Done():
			return imported, ctx.Err()
		case <-time.After(800 * time.Millisecond):
		}
	}
	return imported, nil
}

// importOneChannelFile inserts a metadata-only file + a Telegram version row
// pointing at the channel message. Returns false (no error) if the message is
// already known (was uploaded by this app, or already imported).
func (s *Service) importOneChannelFile(ctx context.Context, folderID string, peer StoragePeer, cf ChannelFile) (bool, error) {
	userID := UserFromContext(ctx)

	// Skip if this channel message is already represented in the drive.
	var exists int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM file_versions
		WHERE telegram_channel_id = ? AND telegram_message_id = ?
		LIMIT 1
	`, peer.ChannelID, cf.MessageID).Scan(&exists)
	if err == nil {
		return false, nil // already known
	}
	if err != sql.ErrNoRows {
		return false, fmt.Errorf("kiểm tra message đã import: %w", err)
	}

	now := time.Now().Unix()
	safeName := filepath.Base(cf.Name)
	if safeName == "" || safeName == "." || safeName == "/" {
		safeName = fmt.Sprintf("telegram-%d", cf.MessageID)
	}
	ext := strings.ToLower(filepath.Ext(safeName))
	mime := cf.MimeType
	if mime == "" {
		mime = detectMimeType(safeName, "", nil)
	}
	kind := classifyKind(mime, ext)
	id := newID()
	versionID := newID()
	fileID := fmt.Sprintf("channel:%d:%d", peer.ChannelID, cf.MessageID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// File row: no local cache, already on Telegram → telegram_synced.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO files (id, folder_id, name, extension, kind, size, mime_type, hash, current_version_id, sync_state, local_path, thumbnail_path, preview_status, cache_origin, user_id, created_at, updated_at)
		VALUES (?, NULLIF(?, ''), ?, ?, ?, ?, ?, '', ?, 'telegram_synced', '', '', ?, ?, NULLIF(?, ''), ?, ?)
	`, id, folderID, safeName, ext, kind, cf.Size, mime, versionID, previewStatusForKind(kind), CacheOriginSync, userID, now, now)
	if err != nil {
		return false, fmt.Errorf("ghi file import: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO file_versions (id, file_id, version_number, size, hash, telegram_channel_id, telegram_message_id, telegram_file_id, telegram_access_hash, created_at)
		VALUES (?, ?, 1, ?, '', NULLIF(?, 0), ?, ?, ?, ?)
	`, versionID, id, cf.Size, peer.ChannelID, cf.MessageID, fileID, peer.AccessHash, now)
	if err != nil {
		return false, fmt.Errorf("ghi version import: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}

	s.events.Publish("file.created", File{ID: id, FolderID: folderID, Name: safeName, Extension: ext, Kind: kind, Size: cf.Size, MimeType: mime, SyncState: "telegram_synced", CreatedAt: now, UpdatedAt: now})
	return true, nil
}
