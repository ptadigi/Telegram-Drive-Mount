package drive

import (
	"archive/zip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SearchResult struct {
	Folders []Folder `json:"folders"`
	Files   []File   `json:"files"`
}

func (s *Service) Search(ctx context.Context, query string) (SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchResult{Folders: []Folder{}, Files: []File{}}, nil
	}
	pattern := "%" + strings.ToLower(query) + "%"
	folders, err := s.searchFolders(ctx, pattern)
	if err != nil {
		return SearchResult{}, err
	}
	files, err := s.searchFiles(ctx, pattern)
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Folders: folders, Files: files}, nil
}

func (s *Service) searchFolders(ctx context.Context, pattern string) ([]Folder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(parent_id, ''), name, created_at, updated_at
		FROM folders
		WHERE deleted_at IS NULL AND LOWER(name) LIKE ?
		ORDER BY updated_at DESC
		LIMIT 200
	`, pattern)
	if err != nil {
		return nil, fmt.Errorf("tìm thư mục: %w", err)
	}
	defer rows.Close()
	folders := make([]Folder, 0)
	for rows.Next() {
		var folder Folder
		if err := rows.Scan(&folder.ID, &folder.ParentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, folder)
	}
	return folders, rows.Err()
}

func (s *Service) searchFiles(ctx context.Context, pattern string) ([]File, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, COALESCE(extension, ''), COALESCE(kind, 'other'), size, COALESCE(mime_type, ''), sync_state, COALESCE(local_path, ''), COALESCE(thumbnail_path, ''), COALESCE(preview_status, 'pending'), created_at, updated_at
		FROM files
		WHERE deleted_at IS NULL AND LOWER(name) LIKE ?
		ORDER BY updated_at DESC
		LIMIT 500
	`, pattern)
	if err != nil {
		return nil, fmt.Errorf("tìm file: %w", err)
	}
	defer rows.Close()
	files := make([]File, 0)
	for rows.Next() {
		var file File
		if err := rows.Scan(&file.ID, &file.FolderID, &file.Name, &file.Extension, &file.Kind, &file.Size, &file.MimeType, &file.SyncState, &file.LocalPath, &file.ThumbnailPath, &file.PreviewStatus, &file.CreatedAt, &file.UpdatedAt); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

type MoveInput struct {
	ID          string `json:"id"`
	NewParentID string `json:"new_parent_id"`
}

func (s *Service) MoveFile(ctx context.Context, input MoveInput) (File, error) {
	if input.ID == "" {
		return File{}, fmt.Errorf("thiếu id file")
	}
	if input.NewParentID != "" {
		if err := s.ensureFolderExists(ctx, input.NewParentID); err != nil {
			return File{}, err
		}
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `UPDATE files SET folder_id = NULLIF(?, ''), updated_at = ? WHERE id = ?`, input.NewParentID, now, input.ID); err != nil {
		return File{}, fmt.Errorf("di chuyển file: %w", err)
	}
	file, err := s.getFile(ctx, input.ID)
	if err != nil {
		return File{}, err
	}
	s.events.Publish("file.updated", file)
	return file, nil
}

func (s *Service) MoveFolder(ctx context.Context, input MoveInput) (Folder, error) {
	if input.ID == "" {
		return Folder{}, fmt.Errorf("thiếu id thư mục")
	}
	if input.NewParentID == input.ID {
		return Folder{}, fmt.Errorf("không thể di chuyển thư mục vào chính nó")
	}
	if input.NewParentID != "" {
		if err := s.ensureFolderExists(ctx, input.NewParentID); err != nil {
			return Folder{}, err
		}
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `UPDATE folders SET parent_id = NULLIF(?, ''), updated_at = ? WHERE id = ?`, input.NewParentID, now, input.ID); err != nil {
		return Folder{}, fmt.Errorf("di chuyển thư mục: %w", err)
	}
	var folder Folder
	if err := s.db.QueryRowContext(ctx, `SELECT id, COALESCE(parent_id, ''), name, created_at, updated_at FROM folders WHERE id = ?`, input.ID).Scan(&folder.ID, &folder.ParentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
		return Folder{}, err
	}
	s.events.Publish("folder.updated", folder)
	return folder, nil
}

func (s *Service) StarFile(ctx context.Context, id string, starred bool) error {
	if id == "" {
		return fmt.Errorf("thiếu id file")
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE files SET starred = ?, updated_at = ? WHERE id = ?`, boolToInt(starred), time.Now().Unix(), id); err != nil {
		return fmt.Errorf("đánh dấu file: %w", err)
	}
	s.events.Publish("file.starred", map[string]any{"id": id, "starred": starred})
	return nil
}

func (s *Service) StarFolder(ctx context.Context, id string, starred bool) error {
	if id == "" {
		return fmt.Errorf("thiếu id thư mục")
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE folders SET starred = ?, updated_at = ? WHERE id = ?`, boolToInt(starred), time.Now().Unix(), id); err != nil {
		return fmt.Errorf("đánh dấu thư mục: %w", err)
	}
	s.events.Publish("folder.starred", map[string]any{"id": id, "starred": starred})
	return nil
}

func (s *Service) ListStarred(ctx context.Context) (FolderContents, error) {
	folders := make([]Folder, 0)
	rows, err := s.db.QueryContext(ctx, `SELECT id, COALESCE(parent_id, ''), name, created_at, updated_at FROM folders WHERE deleted_at IS NULL AND starred = 1 ORDER BY updated_at DESC`)
	if err != nil {
		return FolderContents{}, err
	}
	for rows.Next() {
		var folder Folder
		if err := rows.Scan(&folder.ID, &folder.ParentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			rows.Close()
			return FolderContents{}, err
		}
		folders = append(folders, folder)
	}
	rows.Close()
	files := make([]File, 0)
	frows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, COALESCE(extension, ''), COALESCE(kind, 'other'), size, COALESCE(mime_type, ''), sync_state, COALESCE(local_path, ''), COALESCE(thumbnail_path, ''), COALESCE(preview_status, 'pending'), created_at, updated_at
		FROM files
		WHERE deleted_at IS NULL AND starred = 1
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return FolderContents{}, err
	}
	defer frows.Close()
	for frows.Next() {
		var file File
		if err := frows.Scan(&file.ID, &file.FolderID, &file.Name, &file.Extension, &file.Kind, &file.Size, &file.MimeType, &file.SyncState, &file.LocalPath, &file.ThumbnailPath, &file.PreviewStatus, &file.CreatedAt, &file.UpdatedAt); err != nil {
			return FolderContents{}, err
		}
		files = append(files, file)
	}
	return FolderContents{Folders: folders, Files: files}, frows.Err()
}

func (s *Service) PermanentDeleteFile(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("thiếu id file")
	}
	var localPath, thumbnailPath sql.NullString
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(local_path, ''), COALESCE(thumbnail_path, '') FROM files WHERE id = ?`, id).Scan(&localPath, &thumbnailPath)
	if _, err := s.db.ExecContext(ctx, `DELETE FROM files WHERE id = ?`, id); err != nil {
		return fmt.Errorf("xóa file: %w", err)
	}
	if localPath.Valid && localPath.String != "" {
		_ = os.Remove(localPath.String)
	}
	if thumbnailPath.Valid && thumbnailPath.String != "" {
		_ = os.Remove(thumbnailPath.String)
	}
	s.events.Publish("file.deleted", map[string]any{"id": id})
	return nil
}

func (s *Service) PermanentDeleteFolder(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("thiếu id thư mục")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM files WHERE folder_id = ?`, id)
	if err != nil {
		return err
	}
	var fileIDs []string
	for rows.Next() {
		var fileID string
		if err := rows.Scan(&fileID); err != nil {
			rows.Close()
			return err
		}
		fileIDs = append(fileIDs, fileID)
	}
	rows.Close()
	for _, fileID := range fileIDs {
		_ = s.PermanentDeleteFile(ctx, fileID)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM folders WHERE id = ?`, id); err != nil {
		return err
	}
	s.events.Publish("folder.deleted", map[string]any{"id": id})
	return nil
}

func (s *Service) ZipFolder(ctx context.Context, folderID string, w http.ResponseWriter) error {
	if folderID == "" {
		return fmt.Errorf("thiếu id thư mục")
	}
	if err := s.ensureFolderExists(ctx, folderID); err != nil {
		return err
	}
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()
	return s.writeFolderToZip(ctx, folderID, "", zipWriter)
}

func (s *Service) ZipBundle(ctx context.Context, fileIDs []string, folderIDs []string, w http.ResponseWriter) error {
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()
	for _, id := range fileIDs {
		file, err := s.GetDownloadableFile(ctx, id)
		if err != nil {
			continue
		}
		header, err := zipWriter.Create(file.Name)
		if err != nil {
			return err
		}
		source, err := os.Open(file.LocalPath)
		if err != nil {
			continue
		}
		_, copyErr := io.Copy(header, source)
		source.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	for _, id := range folderIDs {
		var name string
		if err := s.db.QueryRowContext(ctx, `SELECT name FROM folders WHERE id = ?`, id).Scan(&name); err != nil {
			continue
		}
		if err := s.writeFolderToZip(ctx, id, name, zipWriter); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) writeFolderToZip(ctx context.Context, folderID string, prefix string, zipWriter *zip.Writer) error {
	folders, err := s.listFolders(ctx, folderID)
	if err != nil {
		return err
	}
	files, err := s.listFilesInFolder(ctx, folderID)
	if err != nil {
		return err
	}
	for _, file := range files {
		downloadable, err := s.GetDownloadableFile(ctx, file.ID)
		if err != nil {
			continue
		}
		header, err := zipWriter.Create(filepath.ToSlash(filepath.Join(prefix, file.Name)))
		if err != nil {
			return err
		}
		source, err := os.Open(downloadable.LocalPath)
		if err != nil {
			continue
		}
		_, copyErr := io.Copy(header, source)
		source.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	for _, folder := range folders {
		nested := filepath.Join(prefix, folder.Name)
		if err := s.writeFolderToZip(ctx, folder.ID, nested, zipWriter); err != nil {
			return err
		}
	}
	return nil
}
