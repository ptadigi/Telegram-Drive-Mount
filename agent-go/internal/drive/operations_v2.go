package drive

import (
	"archive/zip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
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
	if _, err := s.getFile(ctx, input.ID); err != nil {
		return File{}, err
	}
	if input.NewParentID != "" {
		if err := s.ensureFolderExists(ctx, input.NewParentID); err != nil {
			return File{}, err
		}
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `UPDATE files SET folder_id = NULLIF(?, ''), updated_at = ? WHERE id = ? AND COALESCE(user_id, '') = COALESCE(?, '')`, input.NewParentID, now, input.ID, UserFromContext(ctx)); err != nil {
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
	if err := s.ensureFolderExists(ctx, input.ID); err != nil {
		return Folder{}, err
	}
	if input.NewParentID != "" {
		if err := s.ensureFolderExists(ctx, input.NewParentID); err != nil {
			return Folder{}, err
		}
		if err := s.assertNotDescendant(ctx, input.ID, input.NewParentID); err != nil {
			return Folder{}, err
		}
	}
	now := time.Now().Unix()
	if _, err := s.db.ExecContext(ctx, `UPDATE folders SET parent_id = NULLIF(?, ''), updated_at = ? WHERE id = ? AND COALESCE(user_id, '') = COALESCE(?, '')`, input.NewParentID, now, input.ID, UserFromContext(ctx)); err != nil {
		return Folder{}, fmt.Errorf("di chuyển thư mục: %w", err)
	}
	var folder Folder
	if err := s.db.QueryRowContext(ctx, `SELECT id, COALESCE(parent_id, ''), name, created_at, updated_at FROM folders WHERE id = ? AND COALESCE(user_id, '') = COALESCE(?, '')`, input.ID, UserFromContext(ctx)).Scan(&folder.ID, &folder.ParentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
		return Folder{}, err
	}
	s.events.Publish("folder.updated", folder)
	return folder, nil
}

func (s *Service) assertNotDescendant(ctx context.Context, ancestorID, candidateID string) error {
	current := candidateID
	for i := 0; i < 200 && current != ""; i++ {
		if current == ancestorID {
			return fmt.Errorf("không thể di chuyển thư mục vào chính cây con của nó")
		}
		var parent sql.NullString
		err := s.db.QueryRowContext(ctx, `SELECT parent_id FROM folders WHERE id = ? AND COALESCE(user_id, '') = COALESCE(?, '')`, current, UserFromContext(ctx)).Scan(&parent)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		if !parent.Valid || parent.String == "" {
			return nil
		}
		current = parent.String
	}
	return fmt.Errorf("phát hiện vòng lặp thư mục")
}

func (s *Service) StarFile(ctx context.Context, id string, starred bool) error {
	if id == "" {
		return fmt.Errorf("thiếu id file")
	}
	if _, err := s.getFile(ctx, id); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE files SET starred = ?, updated_at = ? WHERE id = ? AND COALESCE(user_id, '') = COALESCE(?, '')`, boolToInt(starred), time.Now().Unix(), id, UserFromContext(ctx)); err != nil {
		return fmt.Errorf("đánh dấu file: %w", err)
	}
	s.events.Publish("file.starred", map[string]any{"id": id, "starred": starred})
	return nil
}

func (s *Service) StarFolder(ctx context.Context, id string, starred bool) error {
	if id == "" {
		return fmt.Errorf("thiếu id thư mục")
	}
	if err := s.ensureFolderExists(ctx, id); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE folders SET starred = ?, updated_at = ? WHERE id = ? AND COALESCE(user_id, '') = COALESCE(?, '')`, boolToInt(starred), time.Now().Unix(), id, UserFromContext(ctx)); err != nil {
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
	var owner sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT user_id FROM files WHERE id = ?`, id).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("không tìm thấy file")
		}
		return err
	}
	requester := UserFromContext(ctx)
	if requester != "" && owner.Valid && owner.String != "" && owner.String != requester {
		return fmt.Errorf("không có quyền với file này")
	}
	var localPath, thumbnailPath sql.NullString
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(local_path, ''), COALESCE(thumbnail_path, '') FROM files WHERE id = ?`, id).Scan(&localPath, &thumbnailPath)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"shares", "file_versions", "transfers", "sync_entries"} {
		col := "file_id"
		if table == "shares" {
			col = "target_id"
		}
		if table == "sync_entries" {
			col = "remote_file_id"
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE %s = ?`, table, col), id); err != nil {
			return fmt.Errorf("dọn %s: %w", table, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM files WHERE id = ?`, id); err != nil {
		return fmt.Errorf("xóa file: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if localPath.Valid && localPath.String != "" && s.IsLocalPathServable(localPath.String) {
		_ = os.Remove(localPath.String)
	}
	if thumbnailPath.Valid && thumbnailPath.String != "" && s.IsLocalPathServable(thumbnailPath.String) {
		_ = os.Remove(thumbnailPath.String)
	}
	s.events.Publish("file.deleted", map[string]any{"id": id})
	return nil
}

func (s *Service) PermanentDeleteFolder(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("thiếu id thư mục")
	}
	var owner sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT user_id FROM folders WHERE id = ?`, id).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("không tìm thấy thư mục")
		}
		return err
	}
	requester := UserFromContext(ctx)
	if requester != "" && owner.Valid && owner.String != "" && owner.String != requester {
		return fmt.Errorf("không có quyền với thư mục này")
	}
	folderIDs, err := s.collectFolderTreeIDs(ctx, id)
	if err != nil {
		return err
	}
	fileIDs := make([]string, 0)
	for _, fid := range folderIDs {
		ids, err := s.fileIDsInFolder(ctx, fid)
		if err != nil {
			return err
		}
		fileIDs = append(fileIDs, ids...)
	}
	for _, fid := range fileIDs {
		if err := s.PermanentDeleteFile(ctx, fid); err != nil {
			return err
		}
	}
	for i := len(folderIDs) - 1; i >= 0; i-- {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM folders WHERE id = ?`, folderIDs[i]); err != nil {
			return err
		}
	}
	s.events.Publish("folder.deleted", map[string]any{"id": id})
	return nil
}

func (s *Service) collectFolderTreeIDs(ctx context.Context, root string) ([]string, error) {
	out := []string{root}
	queue := []string{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		rows, err := s.db.QueryContext(ctx, `SELECT id FROM folders WHERE COALESCE(parent_id,'') = ?`, current)
		if err != nil {
			return nil, err
		}
		var children []string
		for rows.Next() {
			var cid string
			if err := rows.Scan(&cid); err != nil {
				rows.Close()
				return nil, err
			}
			children = append(children, cid)
		}
		rows.Close()
		out = append(out, children...)
		queue = append(queue, children...)
		if len(out) > 50000 {
			return nil, fmt.Errorf("cây thư mục quá lớn")
		}
	}
	return out, nil
}

func (s *Service) fileIDsInFolder(ctx context.Context, folderID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM files WHERE COALESCE(folder_id,'') = ?`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
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
		entry := sanitizeArchivePath("", file.Name)
		if entry == "" {
			continue
		}
		header, err := zipWriter.Create(entry)
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
		safeName := sanitizeArchivePath("", name)
		if safeName == "" {
			continue
		}
		if err := s.writeFolderToZip(ctx, id, safeName, zipWriter); err != nil {
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
		entry := sanitizeArchivePath(prefix, file.Name)
		if entry == "" {
			continue
		}
		header, err := zipWriter.Create(entry)
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
		nested := sanitizeArchivePath(prefix, folder.Name)
		if nested == "" {
			continue
		}
		if err := s.writeFolderToZip(ctx, folder.ID, nested, zipWriter); err != nil {
			return err
		}
	}
	return nil
}

func sanitizeArchivePath(prefix, name string) string {
	cleaned := strings.ReplaceAll(name, "\\", "/")
	cleaned = strings.TrimPrefix(cleaned, "/")
	for strings.Contains(cleaned, "//") {
		cleaned = strings.ReplaceAll(cleaned, "//", "/")
	}
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return ""
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return ""
		}
		if strings.ContainsRune(segment, 0) {
			return ""
		}
	}
	if prefix == "" {
		return cleaned
	}
	return strings.TrimSuffix(prefix, "/") + "/" + cleaned
}
