package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("tạo thư mục dữ liệu: %w", err)
	}

	dsn := buildDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("mở sqlite: %w", err)
	}
	if err := tunePool(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := applyPragmas(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func buildDSN(path string) string {
	values := url.Values{}
	values.Set("_pragma", "journal_mode(WAL)")
	values.Add("_pragma", "synchronous(NORMAL)")
	values.Add("_pragma", "busy_timeout(8000)")
	values.Add("_pragma", "foreign_keys(ON)")
	values.Add("_pragma", "temp_store(MEMORY)")
	values.Add("_pragma", "cache_size(-32000)")
	values.Add("_pragma", "wal_autocheckpoint(200)")
	return "file:" + path + "?" + values.Encode()
}

func tunePool(db *sql.DB) error {
	maxOpen := runtime.GOMAXPROCS(0)
	if maxOpen < 4 {
		maxOpen = 4
	}
	if maxOpen > 8 {
		maxOpen = 8
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(0)
	return nil
}

func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 8000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA temp_store = MEMORY",
	}
	for _, stmt := range pragmas {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("áp dụng %s: %w", stmt, err)
		}
	}
	return nil
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
		`CREATE TABLE IF NOT EXISTS share_config (
			id TEXT PRIMARY KEY,
			mode TEXT NOT NULL DEFAULT 'lan',
			domain TEXT,
			base_url TEXT,
			local_base_url TEXT,
			port INTEGER NOT NULL DEFAULT 8750,
			gateway_token TEXT,
			health_ok INTEGER NOT NULL DEFAULT 0,
			health_message TEXT,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("chạy migration sqlite: %w", err)
		}
	}
	if err := addColumns(db, "files", map[string]string{
		"extension":        "TEXT NOT NULL DEFAULT ''",
		"kind":             "TEXT NOT NULL DEFAULT 'other'",
		"local_path":       "TEXT NOT NULL DEFAULT ''",
		"thumbnail_path":   "TEXT NOT NULL DEFAULT ''",
		"preview_status":   "TEXT NOT NULL DEFAULT 'pending'",
		"starred":          "INTEGER NOT NULL DEFAULT 0",
		"last_accessed_at": "INTEGER NOT NULL DEFAULT 0",
	}); err != nil {
		return err
	}
	if err := addColumns(db, "folders", map[string]string{
		"starred": "INTEGER NOT NULL DEFAULT 0",
	}); err != nil {
		return err
	}
	if err := addColumns(db, "shares", map[string]string{
		"target_kind":   "TEXT NOT NULL DEFAULT 'file'",
		"target_id":     "TEXT NOT NULL DEFAULT ''",
		"updated_at":    "INTEGER NOT NULL DEFAULT 0",
		"max_downloads": "INTEGER NOT NULL DEFAULT 0",
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
