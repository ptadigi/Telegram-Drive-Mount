package drive

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type RenameInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IDInput struct {
	ID string `json:"id"`
}

func (s *Service) RenameFile(ctx context.Context, input RenameInput) (File, error) {
	name := strings.TrimSpace(input.Name)
	if input.ID == "" || name == "" {
		return File{}, fmt.Errorf("thiếu thông tin đổi tên file")
	}
	if strings.ContainsAny(name, `/\\`) {
		return File{}, fmt.Errorf("tên file không được chứa dấu / hoặc \\")
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `UPDATE files SET name = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, name, now, input.ID); err != nil {
		return File{}, fmt.Errorf("đổi tên file: %w", err)
	}
	file, err := s.getFile(ctx, input.ID)
	if err != nil {
		return File{}, err
	}
	s.events.Publish("file.updated", file)
	return file, nil
}

func (s *Service) RenameFolder(ctx context.Context, input RenameInput) (Folder, error) {
	name := strings.TrimSpace(input.Name)
	if input.ID == "" || name == "" {
		return Folder{}, fmt.Errorf("thiếu thông tin đổi tên thư mục")
	}
	if strings.ContainsAny(name, `/\\`) {
		return Folder{}, fmt.Errorf("tên thư mục không được chứa dấu / hoặc \\")
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `UPDATE folders SET name = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, name, now, input.ID); err != nil {
		return Folder{}, fmt.Errorf("đổi tên thư mục: %w", err)
	}
	var folder Folder
	err := s.db.QueryRowContext(ctx, `SELECT id, COALESCE(parent_id, ''), name, created_at, updated_at FROM folders WHERE id = ?`, input.ID).Scan(&folder.ID, &folder.ParentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt)
	if err != nil {
		return Folder{}, err
	}
	s.events.Publish("folder.updated", folder)
	return folder, nil
}

func (s *Service) TrashFile(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("thiếu id file")
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `UPDATE files SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, id); err != nil {
		return fmt.Errorf("đưa file vào thùng rác: %w", err)
	}
	s.events.Publish("file.trashed", map[string]any{"id": id, "deleted_at": now})
	return nil
}

func (s *Service) TrashFolder(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("thiếu id thư mục")
	}
	now := time.Now().Unix()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE folders SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, id); err != nil {
		return fmt.Errorf("đưa thư mục vào thùng rác: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE files SET deleted_at = ?, updated_at = ? WHERE folder_id = ? AND deleted_at IS NULL`, now, now, id); err != nil {
		return fmt.Errorf("đưa file con vào thùng rác: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.events.Publish("folder.trashed", map[string]any{"id": id, "deleted_at": now})
	return nil
}

func (s *Service) RestoreFile(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("thiếu id file")
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `UPDATE files SET deleted_at = NULL, updated_at = ? WHERE id = ?`, now, id); err != nil {
		return fmt.Errorf("khôi phục file: %w", err)
	}
	s.events.Publish("file.restored", map[string]any{"id": id})
	return nil
}

func (s *Service) RestoreFolder(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("thiếu id thư mục")
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `UPDATE folders SET deleted_at = NULL, updated_at = ? WHERE id = ?`, now, id); err != nil {
		return fmt.Errorf("khôi phục thư mục: %w", err)
	}
	s.events.Publish("folder.restored", map[string]any{"id": id})
	return nil
}

func (s *Service) ListTrash(ctx context.Context) (FolderContents, error) {
	folders, err := s.queryTrashFolders(ctx)
	if err != nil {
		return FolderContents{}, err
	}
	files, err := s.queryTrashFiles(ctx)
	if err != nil {
		return FolderContents{}, err
	}
	return FolderContents{Folders: folders, Files: files}, nil
}

func (s *Service) queryTrashFolders(ctx context.Context) ([]Folder, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, COALESCE(parent_id, ''), name, created_at, updated_at FROM folders WHERE deleted_at IS NOT NULL ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("đọc thư mục thùng rác: %w", err)
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

func (s *Service) queryTrashFiles(ctx context.Context) ([]File, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(folder_id, ''), name, COALESCE(extension, ''), COALESCE(kind, 'other'), size, COALESCE(mime_type, ''), sync_state, COALESCE(local_path, ''), COALESCE(thumbnail_path, ''), COALESCE(preview_status, 'pending'), created_at, updated_at
		FROM files
		WHERE deleted_at IS NOT NULL
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("đọc file thùng rác: %w", err)
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
