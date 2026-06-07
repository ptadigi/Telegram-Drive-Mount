package trash

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMoveToTrashFile(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("recycle bin chỉ test trên các platform hỗ trợ")
	}
	if runtime.GOOS == "darwin" {
		// macOS Finder is not available on headless CI runners; the
		// osascript path needs an interactive Finder session.
		t.Skip("macOS Finder yêu cầu phiên đồ họa, skip trên CI headless")
	}
	if runtime.GOOS == "linux" {
		// CI containers may not have a graphical session / gio.
		if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".local", "share")); err != nil {
			t.Skip("không có thư mục trash chuẩn trên runner")
		}
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(target, []byte("bye"), 0o644); err != nil {
		t.Fatalf("setup file: %v", err)
	}
	if err := MoveToTrash(target); err != nil {
		if errors.Is(err, ErrNotSupported) {
			t.Skipf("platform không hỗ trợ recycle bin: %v", err)
		}
		t.Fatalf("MoveToTrash: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("file vẫn còn sau khi recycle: %v", err)
	}
}

func TestMoveToTrashEmpty(t *testing.T) {
	if err := MoveToTrash(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
}
