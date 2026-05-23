package drive

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
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

type TelegramUploader interface {
	UploadToSavedMessages(ctx context.Context, localPath string, originalName string) (UploadedObject, error)
}

type UploadedObject struct {
	MessageID int
	FileID    string
}

type SyncResult struct {
	Uploaded int    `json:"uploaded"`
	Failed   int    `json:"failed"`
	Message  string `json:"message"`
}

type DownloadableFile struct {
	File
	LocalPath string
}

type pendingFile struct {
	File
	LocalPath string
}

type Service struct {
	db       *sql.DB
	dataDir  string
	uploader TelegramUploader
}

func NewService(db *sql.DB, dataDir string, uploader TelegramUploader) *Service {
	return &Service{db: db, dataDir: dataDir, uploader: uploader}
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

func (s *Service) SaveUploadedFile(ctx context.Context, header *multipart.FileHeader) (File, error) {
	source, err := header.Open()
	if err != nil {
		return File{}, fmt.Errorf("mở file upload: %w", err)
	}
	defer source.Close()

	id := newID()
	storageDir := filepath.Join(s.dataDir, "uploads")
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return File{}, fmt.Errorf("tạo thư mục upload: %w", err)
	}

	targetPath := filepath.Join(storageDir, id+"-"+filepath.Base(header.Filename))
	target, err := os.Create(targetPath)
	if err != nil {
		return File{}, fmt.Errorf("tạo file local: %w", err)
	}
	defer target.Close()

	size, err := io.Copy(target, source)
	if err != nil {
		return File{}, fmt.Errorf("lưu file local: %w", err)
	}

	now := time.Now().Unix()
	mimeType := header.Header.Get("Content-Type")
	syncState := "pending_telegram_upload"
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO files (id, folder_id, name, size, mime_type, hash, sync_state, created_at, updated_at)
		VALUES (?, NULL, ?, ?, ?, '', ?, ?, ?)
	`, id, header.Filename, size, mimeType, syncState, now, now)
	if err != nil {
		return File{}, fmt.Errorf("ghi metadata file: %w", err)
	}

	return File{ID: id, Name: header.Filename, Size: size, MimeType: mimeType, SyncState: syncState, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Service) GetDownloadableFile(ctx context.Context, id string) (DownloadableFile, error) {
	var file File
	err := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, size, COALESCE(mime_type, ''), sync_state, created_at, updated_at
		FROM files
		WHERE id = ? AND deleted_at IS NULL
	`, id).Scan(&file.ID, &file.FolderID, &file.Name, &file.Size, &file.MimeType, &file.SyncState, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return DownloadableFile{}, fmt.Errorf("không tìm thấy file")
		}
		return DownloadableFile{}, fmt.Errorf("đọc metadata file: %w", err)
	}

	localPath := filepath.Join(s.dataDir, "uploads", file.ID+"-"+filepath.Base(file.Name))
	if _, err := os.Stat(localPath); err != nil {
		if os.IsNotExist(err) {
			return DownloadableFile{}, fmt.Errorf("file chưa có cache cục bộ để tải xuống")
		}
		return DownloadableFile{}, fmt.Errorf("kiểm tra cache file: %w", err)
	}

	return DownloadableFile{File: file, LocalPath: localPath}, nil
}

func (s *Service) SyncPendingToTelegram(ctx context.Context) (SyncResult, error) {
	if s.uploader == nil {
		return SyncResult{}, fmt.Errorf("chưa có Telegram uploader")
	}

	pending, err := s.pendingTelegramUploads(ctx)
	if err != nil {
		return SyncResult{}, err
	}
	if len(pending) == 0 {
		return SyncResult{Message: "Không có file nào đang chờ đồng bộ Telegram"}, nil
	}

	result := SyncResult{}
	for _, item := range pending {
		if err := s.markSyncState(ctx, item.ID, "telegram_uploading"); err != nil {
			result.Failed++
			continue
		}

		uploaded, err := s.uploader.UploadToSavedMessages(ctx, item.LocalPath, item.Name)
		if err != nil {
			_ = s.markSyncState(ctx, item.ID, "telegram_upload_failed")
			result.Failed++
			continue
		}

		if err := s.recordTelegramVersion(ctx, item.File, uploaded); err != nil {
			_ = s.markSyncState(ctx, item.ID, "telegram_upload_failed")
			result.Failed++
			continue
		}
		result.Uploaded++
	}

	result.Message = fmt.Sprintf("Đã đồng bộ %d file lên Telegram, lỗi %d file", result.Uploaded, result.Failed)
	return result, nil
}

func (s *Service) pendingTelegramUploads(ctx context.Context) ([]pendingFile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, size, COALESCE(mime_type, ''), sync_state, created_at, updated_at
		FROM files
		WHERE deleted_at IS NULL AND sync_state IN ('pending_telegram_upload', 'telegram_upload_failed')
		ORDER BY updated_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("đọc queue đồng bộ Telegram: %w", err)
	}
	defer rows.Close()

	items := make([]pendingFile, 0)
	for rows.Next() {
		var item pendingFile
		if err := rows.Scan(&item.ID, &item.FolderID, &item.Name, &item.Size, &item.MimeType, &item.SyncState, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("đọc file chờ đồng bộ: %w", err)
		}
		item.LocalPath = filepath.Join(s.dataDir, "uploads", item.ID+"-"+filepath.Base(item.Name))
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) markSyncState(ctx context.Context, fileID string, state string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE files SET sync_state = ?, updated_at = ? WHERE id = ?`, state, time.Now().Unix(), fileID)
	return err
}

func (s *Service) recordTelegramVersion(ctx context.Context, file File, uploaded UploadedObject) error {
	now := time.Now().Unix()
	versionID := newID()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mở transaction metadata: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO file_versions (id, file_id, version_number, size, hash, telegram_channel_id, telegram_message_id, telegram_file_id, created_at)
		VALUES (?, ?, 1, ?, '', NULL, ?, ?, ?)
	`, versionID, file.ID, file.Size, uploaded.MessageID, uploaded.FileID, now)
	if err != nil {
		return fmt.Errorf("ghi version Telegram: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE files SET current_version_id = ?, sync_state = 'telegram_synced', updated_at = ? WHERE id = ?
	`, versionID, now, file.ID)
	if err != nil {
		return fmt.Errorf("cập nhật trạng thái file: %w", err)
	}

	return tx.Commit()
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
