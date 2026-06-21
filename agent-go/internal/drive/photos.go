package drive

import (
	"context"
	"strings"
)

// CameraFolderName is the dedicated Drive folder that phone photo backups land
// in. One per user, created on demand.
const CameraFolderName = "Camera"

// EnsureCameraFolder returns (creating if needed) the per-user "Camera" folder
// at the drive root used for phone photo backups.
func (s *Service) EnsureCameraFolder(ctx context.Context) (Folder, error) {
	return s.getOrCreateFolder(ctx, "", CameraFolderName)
}

// PhotoHashesPresent reports which of the given SHA-256 hashes already exist for
// the current user (any non-deleted file). The PWA calls this before uploading
// so it only sends photos the Drive doesn't already have — incremental,
// bandwidth-friendly backup. Returns a set keyed by hash (lowercase).
func (s *Service) PhotoHashesPresent(ctx context.Context, hashes []string) (map[string]bool, error) {
	present := make(map[string]bool)
	// De-dup + normalize input, cap to a sane batch size.
	seen := make(map[string]struct{}, len(hashes))
	clean := make([]string, 0, len(hashes))
	for _, h := range hashes {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		clean = append(clean, h)
		if len(clean) >= 1000 {
			break
		}
	}
	if len(clean) == 0 {
		return present, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(clean)), ",")
	args := make([]any, 0, len(clean)+1)
	for _, h := range clean {
		args = append(args, h)
	}
	args = append(args, UserFromContext(ctx))
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT LOWER(hash) FROM files
		WHERE hash IN (`+placeholders+`)
		  AND deleted_at IS NULL AND size > 0
		  AND COALESCE(user_id, '') = COALESCE(?, '')
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		present[h] = true
	}
	return present, rows.Err()
}
