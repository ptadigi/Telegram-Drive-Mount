package desktop

import (
	"path/filepath"
	"testing"
)

func TestStoreLoadDefaultsToUnset(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	st := store.Load()
	if st.Mode != string(ModeUnset) {
		t.Fatalf("expected unset, got %q", st.Mode)
	}
}

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	want := State{Mode: string(ModeRemote), ServerURL: "https://drive.example.com", MountPoint: "T:"}
	if err := store.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := store.Load()
	if got.Mode != want.Mode || got.ServerURL != want.ServerURL || got.MountPoint != want.MountPoint {
		t.Fatalf("round trip mismatch: %+v vs %+v", got, want)
	}
}

func TestStoreSaveRejectsEmptyMode(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Save(State{}); err == nil {
		t.Fatal("expected error for empty mode")
	}
}

func TestStoreReset(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.Save(State{Mode: string(ModeLocal)}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if st := store.Load(); st.Mode != string(ModeUnset) {
		t.Fatalf("expected unset after reset, got %q", st.Mode)
	}
	// Reset again should be a no-op, not error.
	if err := store.Reset(); err != nil {
		t.Fatalf("reset idempotent: %v", err)
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"drive.example.com":      "https://drive.example.com",
		"https://drive.x.com/":   "https://drive.x.com",
		"127.0.0.1:8750":         "http://127.0.0.1:8750",
		"localhost:8750":         "http://localhost:8750",
		"http://192.168.1.5:8750": "http://192.168.1.5:8750",
	}
	for in, want := range cases {
		got, err := NormalizeURL(in)
		if err != nil {
			t.Fatalf("NormalizeURL(%q): %v", in, err)
		}
		if got != want {
			t.Fatalf("NormalizeURL(%q)=%q, want %q", in, got, want)
		}
	}
	if _, err := NormalizeURL("  "); err == nil {
		t.Fatal("expected error for empty url")
	}
}

func TestStorePathUnderDataDir(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if store.path != filepath.Join(dir, "desktop.json") {
		t.Fatalf("unexpected store path: %q", store.path)
	}
}
