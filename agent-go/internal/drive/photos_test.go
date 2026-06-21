package drive

import (
	"context"
	"path/filepath"
	"testing"

	"telegram-drive-agent/internal/db"
)

func newPhotoTestService(t *testing.T) *Service {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "photos-test.db")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return NewService(conn, t.TempDir(), nil)
}

func TestEnsureCameraFolderIsIdempotent(t *testing.T) {
	svc := newPhotoTestService(t)
	ctx := WithUser(context.Background(), "user-1")

	first, err := svc.EnsureCameraFolder(ctx)
	if err != nil {
		t.Fatalf("ensure camera folder: %v", err)
	}
	if first.Name != CameraFolderName {
		t.Fatalf("expected folder name %q, got %q", CameraFolderName, first.Name)
	}
	second, err := svc.EnsureCameraFolder(ctx)
	if err != nil {
		t.Fatalf("ensure camera folder (2nd): %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same folder id on repeat, got %q vs %q", first.ID, second.ID)
	}
}

func TestPhotoHashesPresent(t *testing.T) {
	svc := newPhotoTestService(t)
	ctx := WithUser(context.Background(), "user-1")
	now := int64(1700000000)

	// Insert one file with a known hash for user-1.
	if _, err := svc.db.ExecContext(ctx, `
		INSERT INTO files (id, name, size, hash, sync_state, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "f1", "a.jpg", 100, "abc123", "telegram_synced", "user-1", now, now); err != nil {
		t.Fatalf("insert file: %v", err)
	}

	got, err := svc.PhotoHashesPresent(ctx, []string{"ABC123", "deadbeef", "abc123"})
	if err != nil {
		t.Fatalf("hashes present: %v", err)
	}
	if !got["abc123"] {
		t.Fatalf("expected abc123 present (case-insensitive)")
	}
	if got["deadbeef"] {
		t.Fatalf("deadbeef should not be present")
	}
}

func TestPhotoHashesPresentScopedByUser(t *testing.T) {
	svc := newPhotoTestService(t)
	now := int64(1700000000)
	ctxA := WithUser(context.Background(), "user-A")
	ctxB := WithUser(context.Background(), "user-B")

	if _, err := svc.db.ExecContext(ctxA, `
		INSERT INTO files (id, name, size, hash, sync_state, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "fa", "a.jpg", 100, "hashx", "telegram_synced", "user-A", now, now); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// user-A sees it.
	a, err := svc.PhotoHashesPresent(ctxA, []string{"hashx"})
	if err != nil {
		t.Fatalf("present A: %v", err)
	}
	if !a["hashx"] {
		t.Fatalf("user-A should see own hash")
	}
	// user-B must NOT see another user's hash (no cross-user leak).
	b, err := svc.PhotoHashesPresent(ctxB, []string{"hashx"})
	if err != nil {
		t.Fatalf("present B: %v", err)
	}
	if b["hashx"] {
		t.Fatalf("user-B must not see user-A's hash")
	}
}

func TestPhotoHashesPresentIgnoresDeleted(t *testing.T) {
	svc := newPhotoTestService(t)
	ctx := WithUser(context.Background(), "user-1")
	now := int64(1700000000)

	if _, err := svc.db.ExecContext(ctx, `
		INSERT INTO files (id, name, size, hash, sync_state, user_id, created_at, updated_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "fd", "d.jpg", 100, "delhash", "telegram_synced", "user-1", now, now, now); err != nil {
		t.Fatalf("insert deleted: %v", err)
	}
	got, err := svc.PhotoHashesPresent(ctx, []string{"delhash"})
	if err != nil {
		t.Fatalf("present: %v", err)
	}
	if got["delhash"] {
		t.Fatalf("deleted file hash must not count as present")
	}
}
