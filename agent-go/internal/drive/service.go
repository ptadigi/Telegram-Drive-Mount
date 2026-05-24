package drive

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"sync"
	"time"
)

type File struct {
	ID            string `json:"id"`
	FolderID      string `json:"folder_id,omitempty"`
	Name          string `json:"name"`
	Extension     string `json:"extension"`
	Kind          string `json:"kind"`
	Size          int64  `json:"size"`
	MimeType      string `json:"mime_type,omitempty"`
	SyncState     string `json:"sync_state"`
	LocalPath     string `json:"-"`
	ThumbnailPath string `json:"thumbnail_path,omitempty"`
	PreviewStatus string `json:"preview_status"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type Folder struct {
	ID        string `json:"id"`
	ParentID  string `json:"parent_id,omitempty"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type FolderContents struct {
	FolderID string   `json:"folder_id,omitempty"`
	Folders  []Folder `json:"folders"`
	Files    []File   `json:"files"`
}

type CreateFolderInput struct {
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
}

type StoragePeer struct {
	Kind       string
	ChannelID  int64
	AccessHash int64
}

type TelegramUploader interface {
	UploadToSavedMessages(ctx context.Context, localPath string, originalName string) (UploadedObject, error)
	DownloadFromSavedMessages(ctx context.Context, messageID int, targetPath string) error
	StreamFromSavedMessages(ctx context.Context, messageID int, offset int64, length int64, w io.Writer) (StreamResult, error)
}

type StreamResult struct {
	Size     int64
	MimeType string
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

type Transfer struct {
	ID         string `json:"id"`
	FileID     string `json:"file_id"`
	Kind       string `json:"kind"`
	Phase      string `json:"phase"`
	Percent    int    `json:"percent"`
	BytesDone  int64  `json:"bytes_done"`
	BytesTotal int64  `json:"bytes_total"`
	LastError  string `json:"last_error,omitempty"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type DownloadableFile struct {
	File
	LocalPath string
}

type ThumbnailFile struct {
	Path     string
	MimeType string
}

type pendingFile struct {
	File
	LocalPath string
}

type Service struct {
	db           *sql.DB
	dataDir      string
	uploader     TelegramUploader
	events       *EventBus
	tunnelURL    string
	tunnelActive bool
	tunnelMu     sync.RWMutex
	cachePolicy  CachePolicy
	cacheMu      sync.RWMutex
}

func NewService(db *sql.DB, dataDir string, uploader TelegramUploader) *Service {
	return &Service{db: db, dataDir: dataDir, uploader: uploader, events: NewEventBus(), cachePolicy: CachePolicy{Mode: "smart", MaxBytes: 5 * 1024 * 1024 * 1024}}
}

func (s *Service) Events() *EventBus {
	return s.events
}

func (s *Service) SetTunnelStatus(active bool, url string) {
	s.tunnelMu.Lock()
	s.tunnelActive = active
	s.tunnelURL = url
	s.tunnelMu.Unlock()
	cfg, err := s.GetShareConfig(context.Background())
	if err == nil {
		cfg.TunnelActive = active
		cfg.TunnelURL = url
		s.events.Publish("share.config", cfg)
	}
}

func (s *Service) ListFiles(ctx context.Context) ([]File, error) {
	contents, err := s.ListFolderContents(ctx, "")
	if err != nil {
		return nil, err
	}
	return contents.Files, nil
}

func (s *Service) ListFolderContents(ctx context.Context, folderID string) (FolderContents, error) {
	folders, err := s.listFolders(ctx, folderID)
	if err != nil {
		return FolderContents{}, err
	}
	files, err := s.listFilesInFolder(ctx, folderID)
	if err != nil {
		return FolderContents{}, err
	}
	return FolderContents{FolderID: folderID, Folders: folders, Files: files}, nil
}

func (s *Service) CreateFolder(ctx context.Context, input CreateFolderInput) (Folder, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Folder{}, fmt.Errorf("vui lòng nhập tên thư mục")
	}
	if strings.ContainsAny(name, `/\\`) {
		return Folder{}, fmt.Errorf("tên thư mục không được chứa dấu / hoặc \\")
	}
	if input.ParentID != "" {
		if err := s.ensureFolderExists(ctx, input.ParentID); err != nil {
			return Folder{}, err
		}
	}

	now := time.Now().Unix()
	folder := Folder{ID: newID(), ParentID: input.ParentID, Name: name, CreatedAt: now, UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO folders (id, parent_id, name, created_at, updated_at)
		VALUES (?, NULLIF(?, ''), ?, ?, ?)
	`, folder.ID, folder.ParentID, folder.Name, now, now)
	if err != nil {
		return Folder{}, fmt.Errorf("tạo thư mục: %w", err)
	}
	return folder, nil
}

func (s *Service) SaveUploadedFile(ctx context.Context, header *multipart.FileHeader, folderID string, relativePath string) (File, error) {
	source, err := header.Open()
	if err != nil {
		return File{}, fmt.Errorf("mở file upload: %w", err)
	}
	defer source.Close()
	return s.saveFileFromReader(ctx, source, header.Filename, header.Header.Get("Content-Type"), folderID, relativePath, true)
}

func (s *Service) SaveLocalFile(ctx context.Context, sourcePath string, folderID string, relativePath string) (File, error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return File{}, fmt.Errorf("mở file local: %w", err)
	}
	defer source.Close()
	return s.saveFileFromReader(ctx, source, filepath.Base(sourcePath), "", folderID, relativePath, false)
}

func (s *Service) saveFileFromReader(ctx context.Context, source io.Reader, filename string, headerMime string, folderID string, relativePath string, copyToCache bool) (File, error) {
	if folderID != "" {
		if err := s.ensureFolderExists(ctx, folderID); err != nil {
			return File{}, err
		}
	}
	folderID, err := s.ensureRelativeFolderPath(ctx, folderID, relativePath)
	if err != nil {
		return File{}, err
	}

	id := newID()
	storageDir := filepath.Join(s.dataDir, "uploads")
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return File{}, fmt.Errorf("tạo thư mục upload: %w", err)
	}

	safeName := filepath.Base(filename)
	targetPath := filepath.Join(storageDir, id+"-"+safeName)
	target, err := os.Create(targetPath)
	if err != nil {
		return File{}, fmt.Errorf("tạo file local: %w", err)
	}
	defer target.Close()

	hasher := sha256.New()
	teeWriter := io.MultiWriter(target, hasher)

	buffer := make([]byte, 512)
	read, readErr := source.Read(buffer)
	if readErr != nil && readErr != io.EOF {
		return File{}, fmt.Errorf("đọc file upload: %w", readErr)
	}
	mimeType := detectMimeType(safeName, headerMime, buffer[:read])
	if _, err := teeWriter.Write(buffer[:read]); err != nil {
		return File{}, fmt.Errorf("lưu file local: %w", err)
	}
	rest, err := io.Copy(teeWriter, source)
	if err != nil {
		return File{}, fmt.Errorf("lưu file local: %w", err)
	}
	size := int64(read) + rest
	hashHex := hex.EncodeToString(hasher.Sum(nil))
	_ = copyToCache

	if existing, err := s.findFileByHash(ctx, hashHex); err == nil && existing.ID != "" {
		_ = os.Remove(targetPath)
		return existing, nil
	}

	now := time.Now().Unix()
	ext := strings.ToLower(filepath.Ext(safeName))
	kind := classifyKind(mimeType, ext)
	syncState := "pending_telegram_upload"
	thumbnailPath, previewStatus := s.preparePreview(id, targetPath, kind)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return File{}, fmt.Errorf("mở transaction upload: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO files (id, folder_id, name, extension, kind, size, mime_type, hash, current_version_id, sync_state, local_path, thumbnail_path, preview_status, created_at, updated_at)
		VALUES (?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?)
	`, id, folderID, safeName, ext, kind, size, mimeType, hashHex, syncState, targetPath, thumbnailPath, previewStatus, now, now)
	if err != nil {
		return File{}, fmt.Errorf("ghi metadata file: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO transfers (id, file_id, kind, phase, percent, bytes_done, bytes_total, created_at, updated_at)
		VALUES (?, ?, 'telegram_sync', 'queued', 0, 0, ?, ?, ?)
	`, newID(), id, size, now, now); err != nil {
		return File{}, fmt.Errorf("ghi queue đồng bộ: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return File{}, fmt.Errorf("lưu metadata upload: %w", err)
	}

	file := File{ID: id, FolderID: folderID, Name: safeName, Extension: ext, Kind: kind, Size: size, MimeType: mimeType, SyncState: syncState, LocalPath: targetPath, ThumbnailPath: thumbnailPath, PreviewStatus: previewStatus, CreatedAt: now, UpdatedAt: now}
	s.events.Publish("file.created", file)
	return file, nil
}

func (s *Service) findFileByHash(ctx context.Context, hash string) (File, error) {
	if hash == "" {
		return File{}, sql.ErrNoRows
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, COALESCE(extension, ''), COALESCE(kind, 'other'), size, COALESCE(mime_type, ''), sync_state, COALESCE(local_path, ''), COALESCE(thumbnail_path, ''), COALESCE(preview_status, 'pending'), created_at, updated_at
		FROM files
		WHERE hash = ? AND deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT 1
	`, hash)
	var file File
	if err := row.Scan(&file.ID, &file.FolderID, &file.Name, &file.Extension, &file.Kind, &file.Size, &file.MimeType, &file.SyncState, &file.LocalPath, &file.ThumbnailPath, &file.PreviewStatus, &file.CreatedAt, &file.UpdatedAt); err != nil {
		return File{}, err
	}
	return file, nil
}

func (s *Service) GetDownloadableFile(ctx context.Context, id string) (DownloadableFile, error) {
	file, err := s.getFile(ctx, id)
	if err != nil {
		return DownloadableFile{}, err
	}
	localPath := file.LocalPath
	if localPath == "" {
		localPath = filepath.Join(s.dataDir, "uploads", file.ID+"-"+filepath.Base(file.Name))
	}
	if _, err := os.Stat(localPath); err == nil {
		s.RecordFileAccess(ctx, file.ID)
		return DownloadableFile{File: file, LocalPath: localPath}, nil
	}
	if err := s.restoreFromTelegram(ctx, file, localPath); err != nil {
		return DownloadableFile{}, fmt.Errorf("không có cache cục bộ và không tải được từ Telegram: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE files SET local_path = ?, last_accessed_at = ?, updated_at = ? WHERE id = ?`, localPath, time.Now().Unix(), time.Now().Unix(), file.ID); err == nil {
		file.LocalPath = localPath
	}
	return DownloadableFile{File: file, LocalPath: localPath}, nil
}

func (s *Service) restoreFromTelegram(ctx context.Context, file File, localPath string) error {
	if s.uploader == nil {
		return fmt.Errorf("chưa có Telegram uploader")
	}
	messageID, err := s.latestTelegramMessageID(ctx, file.ID)
	if err != nil {
		return err
	}
	if messageID == 0 {
		return fmt.Errorf("file chưa được đồng bộ Telegram")
	}
	return s.uploader.DownloadFromSavedMessages(ctx, messageID, localPath)
}

func (s *Service) latestTelegramMessageID(ctx context.Context, fileID string) (int, error) {
	var messageID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT telegram_message_id FROM file_versions WHERE file_id = ? ORDER BY version_number DESC LIMIT 1`, fileID).Scan(&messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if !messageID.Valid {
		return 0, nil
	}
	return int(messageID.Int64), nil
}

type StreamSource int

const (
	StreamFromCache StreamSource = iota
	StreamFromTelegram
)

type StreamFile struct {
	File
	Size      int64
	MimeType  string
	LocalPath string
	Source    StreamSource
}

func (s *Service) GetStreamableFile(ctx context.Context, id string) (StreamFile, error) {
	file, err := s.getFile(ctx, id)
	if err != nil {
		return StreamFile{}, err
	}
	localPath := file.LocalPath
	if localPath == "" {
		localPath = filepath.Join(s.dataDir, "uploads", file.ID+"-"+filepath.Base(file.Name))
	}
	if info, err := os.Stat(localPath); err == nil {
		s.RecordFileAccess(ctx, file.ID)
		return StreamFile{File: file, Size: info.Size(), MimeType: file.MimeType, LocalPath: localPath, Source: StreamFromCache}, nil
	}
	messageID, err := s.latestTelegramMessageID(ctx, file.ID)
	if err != nil {
		return StreamFile{}, err
	}
	if messageID == 0 {
		return StreamFile{}, fmt.Errorf("file chưa được đồng bộ Telegram")
	}
	return StreamFile{File: file, Size: file.Size, MimeType: file.MimeType, Source: StreamFromTelegram}, nil
}

func (s *Service) StreamFromTelegram(ctx context.Context, fileID string, offset, length int64, w io.Writer) (StreamResult, error) {
	if s.uploader == nil {
		return StreamResult{}, fmt.Errorf("chưa có Telegram uploader")
	}
	messageID, err := s.latestTelegramMessageID(ctx, fileID)
	if err != nil {
		return StreamResult{}, err
	}
	if messageID == 0 {
		return StreamResult{}, fmt.Errorf("file chưa được đồng bộ Telegram")
	}
	uploader, ok := s.uploader.(interface {
		StreamFromSavedMessages(ctx context.Context, messageID int, offset int64, length int64, w io.Writer) (StreamResult, error)
	})
	if !ok {
		return StreamResult{}, fmt.Errorf("uploader không hỗ trợ stream")
	}
	s.RecordFileAccess(ctx, fileID)
	return uploader.StreamFromSavedMessages(ctx, messageID, offset, length, w)
}

func (s *Service) GetThumbnail(ctx context.Context, id string) (ThumbnailFile, error) {
	file, err := s.getFile(ctx, id)
	if err != nil {
		return ThumbnailFile{}, err
	}
	if file.ThumbnailPath == "" {
		return ThumbnailFile{}, fmt.Errorf("file chưa có thumbnail")
	}
	if _, err := os.Stat(file.ThumbnailPath); err != nil {
		return ThumbnailFile{}, fmt.Errorf("thumbnail không còn trong cache: %w", err)
	}
	return ThumbnailFile{Path: file.ThumbnailPath, MimeType: "image/jpeg"}, nil
}

func (s *Service) ListTransfers(ctx context.Context) ([]Transfer, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_id, kind, phase, percent, bytes_done, bytes_total, COALESCE(last_error, ''), created_at, updated_at
		FROM transfers
		WHERE phase NOT IN ('completed') OR updated_at > ?
		ORDER BY updated_at DESC
		LIMIT 50
	`, time.Now().Add(-10*time.Minute).Unix())
	if err != nil {
		return nil, fmt.Errorf("đọc danh sách transfer: %w", err)
	}
	defer rows.Close()
	transfers := make([]Transfer, 0)
	for rows.Next() {
		var transfer Transfer
		if err := rows.Scan(&transfer.ID, &transfer.FileID, &transfer.Kind, &transfer.Phase, &transfer.Percent, &transfer.BytesDone, &transfer.BytesTotal, &transfer.LastError, &transfer.CreatedAt, &transfer.UpdatedAt); err != nil {
			return nil, fmt.Errorf("đọc transfer: %w", err)
		}
		transfers = append(transfers, transfer)
	}
	return transfers, rows.Err()
}

func (s *Service) SyncWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.SyncPendingToTelegram(ctx)
		}
	}
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
		_ = s.updateTransfer(ctx, item.ID, "syncing_telegram", 15, 0, item.Size, "")
		uploaded, err := s.uploader.UploadToSavedMessages(ctx, item.LocalPath, item.Name)
		if err != nil {
			_ = s.markSyncState(ctx, item.ID, "telegram_upload_failed")
			_ = s.updateTransfer(ctx, item.ID, "failed", 100, 0, item.Size, err.Error())
			result.Failed++
			continue
		}
		_ = s.updateTransfer(ctx, item.ID, "syncing_telegram", 90, item.Size, item.Size, "")
		if err := s.recordTelegramVersion(ctx, item.File, uploaded); err != nil {
			_ = s.markSyncState(ctx, item.ID, "telegram_upload_failed")
			_ = s.updateTransfer(ctx, item.ID, "failed", 100, item.Size, item.Size, err.Error())
			result.Failed++
			continue
		}
		_ = s.updateTransfer(ctx, item.ID, "completed", 100, item.Size, item.Size, "")
		result.Uploaded++
	}
	result.Message = fmt.Sprintf("Đã đồng bộ %d file lên Telegram, lỗi %d file", result.Uploaded, result.Failed)
	return result, nil
}

func (s *Service) listFolders(ctx context.Context, parentID string) ([]Folder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(parent_id, ''), name, created_at, updated_at
		FROM folders
		WHERE deleted_at IS NULL AND COALESCE(parent_id, '') = ?
		ORDER BY name ASC
	`, parentID)
	if err != nil {
		return nil, fmt.Errorf("đọc danh sách thư mục: %w", err)
	}
	defer rows.Close()
	folders := make([]Folder, 0)
	for rows.Next() {
		var folder Folder
		if err := rows.Scan(&folder.ID, &folder.ParentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, fmt.Errorf("đọc thư mục: %w", err)
		}
		folders = append(folders, folder)
	}
	return folders, rows.Err()
}

func (s *Service) listFilesInFolder(ctx context.Context, folderID string) ([]File, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, COALESCE(extension, ''), COALESCE(kind, 'other'), size, COALESCE(mime_type, ''), sync_state, COALESCE(local_path, ''), COALESCE(thumbnail_path, ''), COALESCE(preview_status, 'pending'), created_at, updated_at
		FROM files
		WHERE deleted_at IS NULL AND COALESCE(folder_id, '') = ?
		ORDER BY updated_at DESC, name ASC
	`, folderID)
	if err != nil {
		return nil, fmt.Errorf("đọc danh sách file: %w", err)
	}
	defer rows.Close()
	files := make([]File, 0)
	for rows.Next() {
		var file File
		if err := rows.Scan(&file.ID, &file.FolderID, &file.Name, &file.Extension, &file.Kind, &file.Size, &file.MimeType, &file.SyncState, &file.LocalPath, &file.ThumbnailPath, &file.PreviewStatus, &file.CreatedAt, &file.UpdatedAt); err != nil {
			return nil, fmt.Errorf("đọc file: %w", err)
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *Service) getFile(ctx context.Context, id string) (File, error) {
	var file File
	err := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, COALESCE(extension, ''), COALESCE(kind, 'other'), size, COALESCE(mime_type, ''), sync_state, COALESCE(local_path, ''), COALESCE(thumbnail_path, ''), COALESCE(preview_status, 'pending'), created_at, updated_at
		FROM files
		WHERE id = ? AND deleted_at IS NULL
	`, id).Scan(&file.ID, &file.FolderID, &file.Name, &file.Extension, &file.Kind, &file.Size, &file.MimeType, &file.SyncState, &file.LocalPath, &file.ThumbnailPath, &file.PreviewStatus, &file.CreatedAt, &file.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return File{}, fmt.Errorf("không tìm thấy file")
		}
		return File{}, fmt.Errorf("đọc metadata file: %w", err)
	}
	return file, nil
}

func (s *Service) ensureRelativeFolderPath(ctx context.Context, parentID string, relativePath string) (string, error) {
	relativePath = strings.ReplaceAll(strings.TrimSpace(relativePath), "\\", "/")
	relativePath = filepath.ToSlash(relativePath)
	if relativePath == "" {
		return parentID, nil
	}
	dir := path.Dir(relativePath)
	if dir == "." || dir == "/" || dir == "" {
		return parentID, nil
	}
	parts := strings.Split(dir, "/")
	currentParent := parentID
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" || name == "." {
			continue
		}
		folder, err := s.getOrCreateFolder(ctx, currentParent, name)
		if err != nil {
			return "", err
		}
		currentParent = folder.ID
	}
	return currentParent, nil
}

func (s *Service) getOrCreateFolder(ctx context.Context, parentID string, name string) (Folder, error) {
	var folder Folder
	err := s.db.QueryRowContext(ctx, `
		SELECT id, COALESCE(parent_id, ''), name, created_at, updated_at
		FROM folders
		WHERE deleted_at IS NULL AND COALESCE(parent_id, '') = ? AND name = ?
	`, parentID, name).Scan(&folder.ID, &folder.ParentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt)
	if err == nil {
		return folder, nil
	}
	if err != sql.ErrNoRows {
		return Folder{}, fmt.Errorf("tìm thư mục: %w", err)
	}
	now := time.Now().Unix()
	folder = Folder{ID: newID(), ParentID: parentID, Name: name, CreatedAt: now, UpdatedAt: now}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO folders (id, parent_id, name, created_at, updated_at)
		VALUES (?, NULLIF(?, ''), ?, ?, ?)
	`, folder.ID, folder.ParentID, folder.Name, now, now)
	if err != nil {
		return Folder{}, fmt.Errorf("tạo thư mục: %w", err)
	}
	return folder, nil
}

func (s *Service) ensureFolderExists(ctx context.Context, id string) error {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM folders WHERE id = ? AND deleted_at IS NULL`, id).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("không tìm thấy thư mục")
		}
		return fmt.Errorf("kiểm tra thư mục: %w", err)
	}
	return nil
}

func (s *Service) updateTransfer(ctx context.Context, fileID, phase string, percent int, bytesDone, bytesTotal int64, lastError string) error {
	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `
		UPDATE transfers
		SET phase = ?, percent = ?, bytes_done = ?, bytes_total = ?, last_error = NULLIF(?, ''), updated_at = ?
		WHERE file_id = ? AND kind = 'telegram_sync' AND phase NOT IN ('completed')
	`, phase, percent, bytesDone, bytesTotal, lastError, now, fileID)
	if err == nil {
		s.events.Publish("transfer.updated", Transfer{FileID: fileID, Kind: "telegram_sync", Phase: phase, Percent: percent, BytesDone: bytesDone, BytesTotal: bytesTotal, LastError: lastError, UpdatedAt: now})
	}
	return err
}

func (s *Service) pendingTelegramUploads(ctx context.Context) ([]pendingFile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, COALESCE(extension, ''), COALESCE(kind, 'other'), size, COALESCE(mime_type, ''), sync_state, COALESCE(local_path, ''), COALESCE(thumbnail_path, ''), COALESCE(preview_status, 'pending'), created_at, updated_at
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
		if err := rows.Scan(&item.ID, &item.FolderID, &item.Name, &item.Extension, &item.Kind, &item.Size, &item.MimeType, &item.SyncState, &item.LocalPath, &item.ThumbnailPath, &item.PreviewStatus, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("đọc file chờ đồng bộ: %w", err)
		}
		if item.LocalPath == "" {
			item.LocalPath = filepath.Join(s.dataDir, "uploads", item.ID+"-"+filepath.Base(item.Name))
		}
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
		INSERT OR IGNORE INTO files (id, folder_id, name, extension, kind, size, mime_type, hash, sync_state, local_path, thumbnail_path, preview_status, created_at, updated_at)
		VALUES (?, NULL, ?, ?, ?, ?, ?, '', ?, '', '', ?, ?, ?)
	`, "demo-welcome", "Chào mừng đến Ổ Đĩa Cloud Ảo.txt", ".txt", "document", int64(1024), "text/plain", "metadata_only", "ready", now, now)
	if err != nil {
		return fmt.Errorf("tạo file demo: %w", err)
	}
	return nil
}

func (s *Service) preparePreview(fileID, localPath, kind string) (string, string) {
	if kind != "image" {
		return "", previewStatusForKind(kind)
	}
	thumbnailPath, err := s.createImageThumbnail(fileID, localPath)
	if err != nil {
		return "", "failed"
	}
	return thumbnailPath, "ready"
}

func (s *Service) createImageThumbnail(fileID, localPath string) (string, error) {
	source, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer source.Close()

	img, _, err := image.Decode(source)
	if err != nil {
		return "", err
	}
	thumb := resizeNearest(img, 320)

	thumbDir := filepath.Join(s.dataDir, "thumbs")
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return "", err
	}
	thumbPath := filepath.Join(thumbDir, fileID+".jpg")
	target, err := os.Create(thumbPath)
	if err != nil {
		return "", err
	}
	defer target.Close()
	if err := jpeg.Encode(target, thumb, &jpeg.Options{Quality: 82}); err != nil {
		return "", err
	}
	return thumbPath, nil
}

func resizeNearest(src image.Image, maxSize int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return src
	}
	if width <= maxSize && height <= maxSize {
		return src
	}
	scale := float64(maxSize) / float64(width)
	if height > width {
		scale = float64(maxSize) / float64(height)
	}
	dstW := int(float64(width) * scale)
	dstH := int(float64(height) * scale)
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		sy := bounds.Min.Y + y*height/dstH
		for x := 0; x < dstW; x++ {
			sx := bounds.Min.X + x*width/dstW
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func detectMimeType(name, headerType string, sample []byte) string {
	if headerType != "" && headerType != "application/octet-stream" {
		return headerType
	}
	if extType := mime.TypeByExtension(strings.ToLower(filepath.Ext(name))); extType != "" {
		return extType
	}
	if len(sample) > 0 {
		return http.DetectContentType(sample)
	}
	return "application/octet-stream"
}

func classifyKind(mimeType, ext string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case mimeType == "application/pdf" || ext == ".doc" || ext == ".docx" || ext == ".xls" || ext == ".xlsx" || ext == ".ppt" || ext == ".pptx" || ext == ".txt":
		return "document"
	case ext == ".zip" || ext == ".rar" || ext == ".7z" || ext == ".tar" || ext == ".gz":
		return "archive"
	default:
		return "other"
	}
}

func previewStatusForKind(kind string) string {
	if kind == "image" || kind == "video" || kind == "audio" || kind == "document" {
		return "pending"
	}
	return "unsupported"
}
