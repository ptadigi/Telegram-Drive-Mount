package devices

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"telegram-drive-agent/internal/db"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "dev.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return New(conn)
}

func TestPairingCodeFormat(t *testing.T) {
	svc := newSvc(t)
	pc, err := svc.StartPairing(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	// Expect XXXX-XXXX from codeChunkCount=2 chunks of codeChunkSize=4.
	if len(pc.Code) != codeChunkSize*codeChunkCount+1 {
		t.Fatalf("unexpected code length %q", pc.Code)
	}
	if pc.Code[codeChunkSize] != '-' {
		t.Fatalf("expected dash separator: %q", pc.Code)
	}
	if pc.ExpiresAt <= time.Now().Unix() {
		t.Fatalf("expires_at must be in the future")
	}
}

func TestExchangeExpiredCode(t *testing.T) {
	svc := newSvc(t)
	// insert an already-expired code directly
	_, err := svc.db.Exec(`INSERT INTO pairing_codes (code, user_id, expires_at, consumed_at, created_at) VALUES (?, ?, ?, 0, ?)`,
		"EXPD-CODE", "u1", time.Now().Add(-time.Minute).Unix(), time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ExchangeCode(context.Background(), "EXPD-CODE", "dev", "linux", ""); err != ErrPairingExpired {
		t.Fatalf("expected ErrPairingExpired, got %v", err)
	}
}

func TestRevokeUnknownDevice(t *testing.T) {
	svc := newSvc(t)
	if err := svc.RevokeDevice(context.Background(), "u1", "nope"); err != ErrInvalidPayload {
		t.Fatalf("expected ErrInvalidPayload for unknown device, got %v", err)
	}
}

func TestTokenIsHashedAtRest(t *testing.T) {
	svc := newSvc(t)
	pc, err := svc.StartPairing(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	res, err := svc.ExchangeCode(context.Background(), pc.Code, "dev", "linux", "")
	if err != nil {
		t.Fatal(err)
	}
	// Plaintext token must NOT be stored in device_tokens.
	var count int
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM device_tokens WHERE token_hash = ?`, res.Token).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("plaintext token leaked into device_tokens")
	}
	// Hash lookup must succeed.
	if err := svc.db.QueryRow(`SELECT COUNT(*) FROM device_tokens WHERE token_hash = ?`, hashToken(res.Token)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("hashed token not found, count=%d", count)
	}
}

func TestUserScoping(t *testing.T) {
	svc := newSvc(t)
	pc, _ := svc.StartPairing(context.Background(), "owner")
	res, err := svc.ExchangeCode(context.Background(), pc.Code, "dev", "linux", "")
	if err != nil {
		t.Fatal(err)
	}
	// Another user must not see or revoke this device.
	other, _ := svc.ListDevices(context.Background(), "intruder")
	if len(other) != 0 {
		t.Fatalf("intruder should not list owner devices")
	}
	if err := svc.RevokeDevice(context.Background(), "intruder", res.Device.ID); err != ErrInvalidPayload {
		t.Fatalf("intruder must not revoke, got %v", err)
	}
	// Token still valid (not revoked by intruder).
	if _, err := svc.ResolveToken(context.Background(), res.Token); err != nil {
		t.Fatalf("token should still resolve: %v", err)
	}
}