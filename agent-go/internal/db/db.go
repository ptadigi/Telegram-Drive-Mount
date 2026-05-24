package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("tạo thư mục dữ liệu: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("mở sqlite: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS folders (
			id TEXT PRIMARY KEY,
			parent_id TEXT,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS files (
			id TEXT PRIMARY KEY,
			folder_id TEXT,
			name TEXT NOT NULL,
			size INTEGER NOT NULL DEFAULT 0,
			mime_type TEXT,
			hash TEXT,
			current_version_id TEXT,
			sync_state TEXT NOT NULL DEFAULT 'cloud_only',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS file_versions (
			id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL,
			version_number INTEGER NOT NULL,
			size INTEGER NOT NULL DEFAULT 0,
			hash TEXT,
			telegram_channel_id INTEGER,
			telegram_message_id INTEGER,
			telegram_file_id TEXT,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS shares (
			id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			password_hash TEXT,
			expires_at INTEGER,
			revoked INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			last_accessed_at INTEGER,
			access_count INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS sync_roots (
			id TEXT PRIMARY KEY,
			local_path TEXT NOT NULL,
			remote_folder_id TEXT,
			mode TEXT NOT NULL DEFAULT 'upload_only',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sync_entries (
			id TEXT PRIMARY KEY,
			sync_root_id TEXT NOT NULL,
			local_path TEXT NOT NULL,
			remote_file_id TEXT,
			local_hash TEXT,
			remote_hash TEXT,
			local_mtime INTEGER,
			remote_updated_at INTEGER,
			state TEXT NOT NULL DEFAULT 'pending_upload',
			last_error TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS transfers (
			id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			phase TEXT NOT NULL,
			percent INTEGER NOT NULL DEFAULT 0,
			bytes_done INTEGER NOT NULL DEFAULT 0,
			bytes_total INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("chạy migration sqlite: %w", err)
		}
	}
	if err := addColumns(db, "files", map[string]string{
		"extension":      "TEXT NOT NULL DEFAULT ''",
		"kind":           "TEXT NOT NULL DEFAULT 'other'",
		"local_path":     "TEXT NOT NULL DEFAULT ''",
		"thumbnail_path": "TEXT NOT NULL DEFAULT ''",
		"preview_status": "TEXT NOT NULL DEFAULT 'pending'",
	}); err != nil {
		return err
	}
	return addColumns(db, "sync_roots", map[string]string{
		"status":       "TEXT NOT NULL DEFAULT 'idle'",
		"last_scan_at": "INTEGER NOT NULL DEFAULT 0",
	})
}

func addColumns(db *sql.DB, table string, columns map[string]string) error {
	existing := map[string]bool{}
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		existing[name] = true
	}
	for name, definition := range columns {
		if existing[name] {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, name, definition)); err != nil {
			return fmt.Errorf("thêm cột %s.%s: %w", table, name, err)
		}
	}
	return nil
}
