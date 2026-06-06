package drive

import (
	"path/filepath"
	"testing"
)

func TestPathInsideCache(t *testing.T) {
	tmp := t.TempDir()
	svc := &Service{dataDir: tmp}
	cacheFile := filepath.Join(tmp, "uploads", "file.bin")
	if !svc.pathInsideCache(cacheFile) {
		t.Fatalf("expected file inside uploads to count as cache")
	}
	outside := filepath.Join(tmp, "..", "outside.txt")
	if svc.pathInsideCache(outside) {
		t.Fatalf("expected file outside dataDir to NOT count as cache")
	}
	if svc.pathInsideCache("") {
		t.Fatalf("expected empty path to be rejected")
	}
}
