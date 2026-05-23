package drive

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type File struct {
	ID        string `json:"id"`
	FolderID  string `json:"folder_id,omitempty"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type,omitempty"`
	SyncState string `json:"sync_state"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListFiles(ctx context.Context) ([]File, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, size, COALESCE(mime_type, ''), sync_state, created_at, updated_at
		FROM files
		WHERE deleted_at IS NULL
		ORDER BY updated_at DESC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("đọc danh sách file: %w", err)
	}
	defer rows.Close()

	files := make([]File, 0)
	for rows.Next() {
		var file File
		if err := rows.Scan(&file.ID, &file.FolderID, &file.Name, &file.Size, &file.MimeType, &file.SyncState, &file.CreatedAt, &file.UpdatedAt); err != nil {
			return nil, fmt.Errorf("đọc file: %w", err)
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *Service) SeedDemoFile(ctx context.Context) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO files (id, folder_id, name, size, mime_type, hash, sync_state, created_at, updated_at)
		VALUES (?, NULL, ?, ?, ?, '', ?, ?, ?)
	`, "demo-welcome", "Chào mừng đến Ổ Đĩa Cloud Ảo.txt", int64(1024), "text/plain", "metadata_only", now, now)
	if err != nil {
		return fmt.Errorf("tạo file demo: %w", err)
	}
	return nil
}
