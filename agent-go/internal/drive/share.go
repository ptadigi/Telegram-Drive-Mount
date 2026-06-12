package drive

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Share struct {
	ID             string `json:"id"`
	Slug           string `json:"slug"`
	TargetKind     string `json:"target_kind"`
	TargetID       string `json:"target_id"`
	HasPassword    bool   `json:"has_password"`
	ExpiresAt      int64  `json:"expires_at,omitempty"`
	Revoked        bool   `json:"revoked"`
	MaxDownloads   int    `json:"max_downloads"`
	AccessCount    int    `json:"access_count"`
	LastAccessedAt int64  `json:"last_accessed_at,omitempty"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
}

type CreateShareInput struct {
	TargetKind   string `json:"target_kind"`
	TargetID     string `json:"target_id"`
	Password     string `json:"password"`
	ExpiresIn    int64  `json:"expires_in"`
	MaxDownloads int    `json:"max_downloads"`
}

type UpdateShareInput struct {
	ID           string  `json:"id"`
	Password     *string `json:"password"`
	ExpiresIn    *int64  `json:"expires_in"`
	Revoked      *bool   `json:"revoked"`
	MaxDownloads *int    `json:"max_downloads"`
}

func (s *Service) CreateShare(ctx context.Context, input CreateShareInput) (Share, error) {
	kind := strings.ToLower(strings.TrimSpace(input.TargetKind))
	if kind != "file" && kind != "folder" {
		return Share{}, fmt.Errorf("loại đối tượng chia sẻ không hợp lệ")
	}
	if strings.TrimSpace(input.TargetID) == "" {
		return Share{}, fmt.Errorf("thiếu id đối tượng chia sẻ")
	}
	if kind == "file" {
		if _, err := s.getFile(ctx, input.TargetID); err != nil {
			return Share{}, err
		}
	} else {
		if err := s.ensureFolderExists(ctx, input.TargetID); err != nil {
			return Share{}, err
		}
	}

	slug, err := generateSlug()
	if err != nil {
		return Share{}, err
	}
	now := time.Now().Unix()
	share := Share{ID: newID(), Slug: slug, TargetKind: kind, TargetID: input.TargetID, MaxDownloads: input.MaxDownloads, CreatedAt: now, UpdatedAt: now}
	if input.ExpiresIn > 0 {
		share.ExpiresAt = now + input.ExpiresIn
	}
	var passwordHash string
	if password := strings.TrimSpace(input.Password); password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return Share{}, fmt.Errorf("tạo mật khẩu chia sẻ: %w", err)
		}
		passwordHash = string(hash)
		share.HasPassword = true
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO shares (id, file_id, slug, password_hash, expires_at, revoked, created_at, last_accessed_at, access_count, target_kind, target_id, updated_at, max_downloads)
		VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, 0), 0, ?, NULL, 0, ?, ?, ?, ?)
	`, share.ID, ifFile(kind, input.TargetID), share.Slug, passwordHash, share.ExpiresAt, now, kind, share.TargetID, now, share.MaxDownloads)
	if err != nil {
		return Share{}, fmt.Errorf("ghi share: %w", err)
	}
	s.events.Publish("share.created", share)
	return share, nil
}

func (s *Service) UpdateShare(ctx context.Context, input UpdateShareInput) (Share, error) {
	if input.ID == "" {
		return Share{}, fmt.Errorf("thiếu id share")
	}
	share, err := s.getShareByID(ctx, input.ID)
	if err != nil {
		return Share{}, err
	}
	now := time.Now().Unix()
	password := share.HasPassword
	passwordHash := ""
	if input.Password != nil {
		if strings.TrimSpace(*input.Password) == "" {
			password = false
			passwordHash = ""
		} else {
			hash, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
			if err != nil {
				return Share{}, err
			}
			passwordHash = string(hash)
			password = true
		}
	}
	expiresAt := share.ExpiresAt
	if input.ExpiresIn != nil {
		if *input.ExpiresIn > 0 {
			expiresAt = now + *input.ExpiresIn
		} else {
			expiresAt = 0
		}
	}
	revoked := share.Revoked
	if input.Revoked != nil {
		revoked = *input.Revoked
	}
	maxDownloads := share.MaxDownloads
	if input.MaxDownloads != nil {
		maxDownloads = *input.MaxDownloads
	}
	args := []any{}
	stmt := `UPDATE shares SET revoked = ?, expires_at = NULLIF(?, 0), updated_at = ?, max_downloads = ?`
	args = append(args, boolToInt(revoked), expiresAt, now, maxDownloads)
	if input.Password != nil {
		stmt += `, password_hash = NULLIF(?, '')`
		args = append(args, passwordHash)
	}
	stmt += ` WHERE id = ?`
	args = append(args, share.ID)
	if _, err := s.db.ExecContext(ctx, stmt, args...); err != nil {
		return Share{}, err
	}
	share.HasPassword = password
	share.ExpiresAt = expiresAt
	share.Revoked = revoked
	share.MaxDownloads = maxDownloads
	share.UpdatedAt = now
	s.events.Publish("share.updated", share)
	return share, nil
}

func (s *Service) DeleteShare(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("thiếu id share")
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM shares WHERE id = ?`, id); err != nil {
		return fmt.Errorf("xóa share: %w", err)
	}
	s.events.Publish("share.deleted", map[string]any{"id": id})
	return nil
}

func (s *Service) ListShares(ctx context.Context, targetKind string, targetID string) ([]Share, error) {
	kind := strings.TrimSpace(targetKind)
	id := strings.TrimSpace(targetID)
	stmt := `SELECT id, COALESCE(slug, ''), COALESCE(target_kind, 'file'), COALESCE(target_id, COALESCE(file_id, '')), CASE WHEN password_hash IS NULL OR password_hash = '' THEN 0 ELSE 1 END, COALESCE(expires_at, 0), revoked, COALESCE(max_downloads, 0), access_count, COALESCE(last_accessed_at, 0), created_at, COALESCE(updated_at, created_at) FROM shares WHERE 1=1`
	args := []any{}
	if kind != "" {
		stmt += ` AND COALESCE(target_kind, 'file') = ?`
		args = append(args, kind)
	}
	if id != "" {
		stmt += ` AND COALESCE(target_id, COALESCE(file_id, '')) = ?`
		args = append(args, id)
	}
	stmt += ` ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("đọc danh sách share: %w", err)
	}
	defer rows.Close()
	shares := make([]Share, 0)
	for rows.Next() {
		var share Share
		var hasPassword, revoked int
		if err := rows.Scan(&share.ID, &share.Slug, &share.TargetKind, &share.TargetID, &hasPassword, &share.ExpiresAt, &revoked, &share.MaxDownloads, &share.AccessCount, &share.LastAccessedAt, &share.CreatedAt, &share.UpdatedAt); err != nil {
			return nil, err
		}
		share.HasPassword = hasPassword == 1
		share.Revoked = revoked == 1
		shares = append(shares, share)
	}
	return shares, rows.Err()
}

func (s *Service) getShareByID(ctx context.Context, id string) (Share, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, COALESCE(slug, ''), COALESCE(target_kind, 'file'), COALESCE(target_id, COALESCE(file_id, '')), CASE WHEN password_hash IS NULL OR password_hash = '' THEN 0 ELSE 1 END, COALESCE(expires_at, 0), revoked, COALESCE(max_downloads, 0), access_count, COALESCE(last_accessed_at, 0), created_at, COALESCE(updated_at, created_at) FROM shares WHERE id = ?`, id)
	var share Share
	var hasPassword, revoked int
	if err := row.Scan(&share.ID, &share.Slug, &share.TargetKind, &share.TargetID, &hasPassword, &share.ExpiresAt, &revoked, &share.MaxDownloads, &share.AccessCount, &share.LastAccessedAt, &share.CreatedAt, &share.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return Share{}, fmt.Errorf("không tìm thấy link chia sẻ")
		}
		return Share{}, err
	}
	share.HasPassword = hasPassword == 1
	share.Revoked = revoked == 1
	return share, nil
}

func ifFile(kind, id string) string {
	if kind == "file" {
		return id
	}
	return ""
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type ResolvedShare struct {
	Share Share
	File  *File
}

func (s *Service) ResolveShare(ctx context.Context, slug string, password string) (ResolvedShare, error) {
	share, err := s.getShareBySlug(ctx, slug)
	if err != nil {
		return ResolvedShare{}, err
	}
	if share.Revoked {
		return ResolvedShare{}, fmt.Errorf("link chia sẻ đã bị thu hồi")
	}
	if share.ExpiresAt > 0 && share.ExpiresAt < time.Now().Unix() {
		return ResolvedShare{}, fmt.Errorf("link chia sẻ đã hết hạn")
	}
	if share.MaxDownloads > 0 && share.AccessCount >= share.MaxDownloads {
		return ResolvedShare{}, fmt.Errorf("link chia sẻ đã đạt giới hạn lượt tải")
	}
	if share.HasPassword {
		hash, err := s.getSharePasswordHash(ctx, share.ID)
		if err != nil {
			return ResolvedShare{}, err
		}
		if password == "" {
			return ResolvedShare{Share: share}, errSharePasswordRequired
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
			return ResolvedShare{}, fmt.Errorf("mật khẩu chia sẻ không đúng")
		}
	}
	resolved := ResolvedShare{Share: share}
	if share.TargetKind == "file" {
		ownerID, err := s.fileOwner(ctx, share.TargetID)
		if err != nil {
			return ResolvedShare{}, err
		}
		shareCtx := WithUser(ctx, ownerID)
		file, err := s.getFile(shareCtx, share.TargetID)
		if err != nil {
			return ResolvedShare{}, err
		}
		resolved.File = &file
	}
	return resolved, nil
}

func (s *Service) fileOwner(ctx context.Context, fileID string) (string, error) {
	var owner sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM files WHERE id = ? AND deleted_at IS NULL`, fileID).Scan(&owner)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("không tìm thấy file")
		}
		return "", err
	}
	return owner.String, nil
}

func (s *Service) RecordShareAccess(ctx context.Context, id string) error {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `
		UPDATE shares
		SET access_count = access_count + 1, last_accessed_at = ?, updated_at = ?
		WHERE id = ? AND revoked = 0 AND (COALESCE(expires_at, 0) = 0 OR COALESCE(expires_at, 0) > ?) AND (COALESCE(max_downloads, 0) = 0 OR access_count < max_downloads)
	`, now, now, id, now)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("link chia sẻ đã đạt giới hạn lượt tải hoặc đã hết hạn")
	}
	return nil
}

func (s *Service) getShareBySlug(ctx context.Context, slug string) (Share, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, COALESCE(slug, ''), COALESCE(target_kind, 'file'), COALESCE(target_id, COALESCE(file_id, '')), CASE WHEN password_hash IS NULL OR password_hash = '' THEN 0 ELSE 1 END, COALESCE(expires_at, 0), revoked, COALESCE(max_downloads, 0), access_count, COALESCE(last_accessed_at, 0), created_at, COALESCE(updated_at, created_at) FROM shares WHERE slug = ?`, slug)
	var share Share
	var hasPassword, revoked int
	if err := row.Scan(&share.ID, &share.Slug, &share.TargetKind, &share.TargetID, &hasPassword, &share.ExpiresAt, &revoked, &share.MaxDownloads, &share.AccessCount, &share.LastAccessedAt, &share.CreatedAt, &share.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return Share{}, fmt.Errorf("không tìm thấy link chia sẻ")
		}
		return Share{}, err
	}
	share.HasPassword = hasPassword == 1
	share.Revoked = revoked == 1
	return share, nil
}

func (s *Service) getSharePasswordHash(ctx context.Context, id string) (string, error) {
	var hash sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM shares WHERE id = ?`, id).Scan(&hash); err != nil {
		return "", err
	}
	return hash.String, nil
}

var errSharePasswordRequired = fmt.Errorf("link chia sẻ yêu cầu mật khẩu")

func IsSharePasswordRequired(err error) bool {
	return err == errSharePasswordRequired
}

func (s *Service) DownloadableForShare(ctx context.Context, share Share) (DownloadableFile, error) {
	if share.TargetKind != "file" {
		return DownloadableFile{}, fmt.Errorf("link chia sẻ này không phải file đơn lẻ")
	}
	owner, err := s.fileOwner(ctx, share.TargetID)
	if err != nil {
		return DownloadableFile{}, err
	}
	return s.GetDownloadableFile(WithUser(ctx, owner), share.TargetID)
}

func (s *Service) StreamFolderShareZip(ctx context.Context, share Share, w http.ResponseWriter) error {
	if share.TargetKind != "folder" {
		return fmt.Errorf("link chia sẻ không phải thư mục")
	}
	owner, err := s.folderOwner(ctx, share.TargetID)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''share.zip")
	return s.ZipFolder(WithUser(ctx, owner), share.TargetID, w)
}

func (s *Service) folderOwner(ctx context.Context, folderID string) (string, error) {
	var owner sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT user_id FROM folders WHERE id = ? AND deleted_at IS NULL`, folderID).Scan(&owner)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("không tìm thấy thư mục")
		}
		return "", err
	}
	return owner.String, nil
}

func generateSlug() (string, error) {
	buf := make([]byte, 9)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	slug := strings.ToLower(strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "="))
	return slug, nil
}
