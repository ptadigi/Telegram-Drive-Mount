package drive

import (
	"context"
	"fmt"
	"time"
)

// ensureShareAccessTable creates the per-access log table once.
func (s *Service) ensureShareAccessTable(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS share_access_log (
			id TEXT PRIMARY KEY,
			share_id TEXT NOT NULL,
			action TEXT NOT NULL,
			ip TEXT,
			user_agent TEXT,
			referer TEXT,
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_share_access_share ON share_access_log(share_id, created_at)`,
	}
	for _, q := range stmts {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// LogShareAccess records one view/download of a share. Best-effort: callers
// run it async and ignore errors so the public share response stays fast.
// action is "view" or "download".
func (s *Service) LogShareAccess(shareID, action, ip, userAgent, referer string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.ensureShareAccessTable(ctx); err != nil {
		return
	}
	if len(userAgent) > 300 {
		userAgent = userAgent[:300]
	}
	if len(referer) > 300 {
		referer = referer[:300]
	}
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO share_access_log (id, share_id, action, ip, user_agent, referer, created_at)
		VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?)
	`, newID(), shareID, action, ip, userAgent, referer, time.Now().Unix())
	// Trim to the most recent 1000 rows per share to bound growth.
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM share_access_log
		WHERE share_id = ? AND id NOT IN (
			SELECT id FROM share_access_log WHERE share_id = ? ORDER BY created_at DESC LIMIT 1000
		)
	`, shareID, shareID)
}

// ShareAccessEntry is one row of a share's access log.
type ShareAccessEntry struct {
	Action    string `json:"action"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Referer   string `json:"referer,omitempty"`
	CreatedAt int64  `json:"created_at"`
}

// ShareAccessStats is the tracking summary + recent log for one share, scoped
// to the requesting user (only the share owner can read it).
type ShareAccessStats struct {
	ShareID   string             `json:"share_id"`
	Views     int                `json:"views"`
	Downloads int                `json:"downloads"`
	Recent    []ShareAccessEntry `json:"recent"`
}

// GetShareAccess returns the tracking log for a share the current user owns.
func (s *Service) GetShareAccess(ctx context.Context, shareID string, limit int) (ShareAccessStats, error) {
	if err := s.ensureShareAccessTable(ctx); err != nil {
		return ShareAccessStats{}, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	// Ownership check: the share's target must belong to the requesting user.
	if err := s.assertShareOwner(ctx, shareID); err != nil {
		return ShareAccessStats{}, err
	}
	out := ShareAccessStats{ShareID: shareID, Recent: []ShareAccessEntry{}}
	// counts
	rows, err := s.db.QueryContext(ctx, `SELECT action, COUNT(*) FROM share_access_log WHERE share_id = ? GROUP BY action`, shareID)
	if err != nil {
		return ShareAccessStats{}, err
	}
	for rows.Next() {
		var action string
		var n int
		if err := rows.Scan(&action, &n); err != nil {
			rows.Close()
			return ShareAccessStats{}, err
		}
		switch action {
		case "download":
			out.Downloads = n
		default:
			out.Views += n
		}
	}
	rows.Close()
	// recent
	r2, err := s.db.QueryContext(ctx, `SELECT action, COALESCE(ip,''), COALESCE(user_agent,''), COALESCE(referer,''), created_at FROM share_access_log WHERE share_id = ? ORDER BY created_at DESC LIMIT ?`, shareID, limit)
	if err != nil {
		return ShareAccessStats{}, err
	}
	defer r2.Close()
	for r2.Next() {
		var e ShareAccessEntry
		if err := r2.Scan(&e.Action, &e.IP, &e.UserAgent, &e.Referer, &e.CreatedAt); err != nil {
			return ShareAccessStats{}, err
		}
		out.Recent = append(out.Recent, e)
	}
	return out, r2.Err()
}

// assertShareOwner returns an error unless the share's target file/folder
// belongs to the user in ctx.
func (s *Service) assertShareOwner(ctx context.Context, shareID string) error {
	uid := UserFromContext(ctx)
	var kind, target string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(target_kind,'file'), COALESCE(target_id, COALESCE(file_id,'')) FROM shares WHERE id = ?`, shareID).Scan(&kind, &target)
	if err != nil {
		return fmt.Errorf("không tìm thấy link chia sẻ")
	}
	var owner string
	tbl := "files"
	if kind == "folder" {
		tbl = "folders"
	}
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COALESCE(user_id,'') FROM %s WHERE id = ?`, tbl), target).Scan(&owner); err != nil {
		return fmt.Errorf("không xác định được chủ sở hữu")
	}
	if owner != uid {
		return fmt.Errorf("không có quyền xem link chia sẻ này")
	}
	return nil
}

// ListSharesForUser returns every share whose target file/folder belongs to the
// current user, newest first, with the target name attached.
func (s *Service) ListSharesForUser(ctx context.Context) ([]ShareWithTarget, error) {
	uid := UserFromContext(ctx)
	rows, err := s.db.QueryContext(ctx, `
		SELECT sh.id, COALESCE(sh.slug,''), COALESCE(sh.target_kind,'file'), COALESCE(sh.target_id, COALESCE(sh.file_id,'')),
		       CASE WHEN sh.password_hash IS NULL OR sh.password_hash='' THEN 0 ELSE 1 END,
		       COALESCE(sh.expires_at,0), sh.revoked, COALESCE(sh.max_downloads,0), sh.access_count,
		       COALESCE(sh.last_accessed_at,0), sh.created_at, COALESCE(sh.updated_at, sh.created_at),
		       COALESCE(f.name, fo.name, '')
		FROM shares sh
		LEFT JOIN files   f  ON sh.target_kind='file'   AND f.id  = COALESCE(sh.target_id, sh.file_id)
		LEFT JOIN folders fo ON sh.target_kind='folder' AND fo.id = sh.target_id
		WHERE COALESCE(f.user_id, fo.user_id, '') = COALESCE(?, '')
		ORDER BY sh.created_at DESC
	`, uid)
	if err != nil {
		return nil, fmt.Errorf("đọc danh sách chia sẻ: %w", err)
	}
	defer rows.Close()
	out := make([]ShareWithTarget, 0)
	for rows.Next() {
		var s2 ShareWithTarget
		var hasPassword, revoked int
		if err := rows.Scan(&s2.ID, &s2.Slug, &s2.TargetKind, &s2.TargetID, &hasPassword, &s2.ExpiresAt, &revoked, &s2.MaxDownloads, &s2.AccessCount, &s2.LastAccessedAt, &s2.CreatedAt, &s2.UpdatedAt, &s2.TargetName); err != nil {
			return nil, err
		}
		s2.HasPassword = hasPassword == 1
		s2.Revoked = revoked == 1
		out = append(out, s2)
	}
	return out, rows.Err()
}

// ShareWithTarget is a share plus the display name of its target.
type ShareWithTarget struct {
	Share
	TargetName string `json:"target_name"`
}
