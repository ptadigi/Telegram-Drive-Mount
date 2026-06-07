package devices

import (
	"context"
	"path/filepath"
	"testing"

	"telegram-drive-agent/internal/db"
)

func setupDB(t *testing.T) (*Service, func()) {
	t.Helper()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	conn, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return New(conn), func() { conn.Close() }
}

func TestPairingFullFlow(t *testing.T) {
	svc, cleanup := setupDB(t)
	defer cleanup()
	ctx := context.Background()

	pc, err := svc.StartPairing(ctx, "user-1")
	if err != nil {
		t.Fatalf("StartPairing: %v", err)
	}
	if pc.Code == "" || pc.ExpiresAt == 0 {
		t.Fatalf("empty pairing code")
	}

	res, err := svc.ExchangeCode(ctx, pc.Code, "Laptop test", "windows", "127.0.0.1")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}
	if res.Token == "" || res.Device.ID == "" || res.Device.UserID != "user-1" {
		t.Fatalf("bad pairing result: %+v", res)
	}

	if _, err := svc.ExchangeCode(ctx, pc.Code, "x", "x", "x"); err != ErrPairingConsumed {
		t.Fatalf("expected ErrPairingConsumed, got %v", err)
	}

	dev, err := svc.ResolveToken(ctx, res.Token)
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if dev.ID != res.Device.ID {
		t.Fatalf("device id mismatch: %s vs %s", dev.ID, res.Device.ID)
	}

	if _, err := svc.ResolveToken(ctx, "tdd1_garbage"); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for unknown token, got %v", err)
	}

	devs, err := svc.ListDevices(ctx, "user-1")
	if err != nil || len(devs) != 1 {
		t.Fatalf("ListDevices got %d devices err=%v", len(devs), err)
	}

	if err := svc.RevokeDevice(ctx, "user-1", dev.ID); err != nil {
		t.Fatalf("RevokeDevice: %v", err)
	}
	if _, err := svc.ResolveToken(ctx, res.Token); err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken after revoke, got %v", err)
	}
}

func TestExchangeCodeNotFound(t *testing.T) {
	svc, cleanup := setupDB(t)
	defer cleanup()
	if _, err := svc.ExchangeCode(context.Background(), "ZZZZ-ZZZZ", "x", "linux", ""); err != ErrPairingNotFound {
		t.Fatalf("expected ErrPairingNotFound, got %v", err)
	}
}

func TestNormalizeCode(t *testing.T) {
	if got := normalizeCode(" 4f2a 9k2x "); got != "4F2A-9K2X" {
		t.Fatalf("normalize: %q", got)
	}
	if got := normalizeCode("4f2a-9k2x"); got != "4F2A-9K2X" {
		t.Fatalf("normalize: %q", got)
	}
}