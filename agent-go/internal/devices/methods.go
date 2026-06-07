package devices

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func (s *Service) StartPairing(ctx context.Context, userID string) (PairingCode, error) {
	if userID == "" {
		return PairingCode{}, ErrInvalidPayload
	}
	code, err := newPairingCode()
	if err != nil {
		return PairingCode{}, err
	}
	now := time.Now()
	expires := now.Add(codeTTL)
	_, err = s.db.ExecContext(ctx, `INSERT INTO pairing_codes (code, user_id, expires_at, consumed_at, device_id, created_at) VALUES (?, ?, ?, 0, NULL, ?)`, code, userID, expires.Unix(), now.Unix())
	if err != nil {
		return PairingCode{}, fmt.Errorf("ghi pairing code: %w", err)
	}
	return PairingCode{Code: code, ExpiresAt: expires.Unix()}, nil
}

func (s *Service) ExchangeCode(ctx context.Context, rawCode, name, platform, lastIP string) (PairingResult, error) {
	code := normalizeCode(rawCode)
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Thiet bi"
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}
	platform = strings.TrimSpace(platform)
	if len(platform) > maxPlatformLen {
		platform = platform[:maxPlatformLen]
	}
	if code == "" {
		return PairingResult{}, ErrInvalidPayload
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PairingResult{}, err
	}
	defer tx.Rollback()

	var userID string
	var expiresAt, consumedAt int64
	row := tx.QueryRowContext(ctx, `SELECT user_id, expires_at, consumed_at FROM pairing_codes WHERE code = ?`, code)
	if err := row.Scan(&userID, &expiresAt, &consumedAt); err != nil {
		if err == sql.ErrNoRows {
			return PairingResult{}, ErrPairingNotFound
		}
		return PairingResult{}, err
	}
	now := time.Now().Unix()
	if consumedAt != 0 {
		return PairingResult{}, ErrPairingConsumed
	}
	if expiresAt > 0 && expiresAt < now {
		return PairingResult{}, ErrPairingExpired
	}

	deviceID := newID()
	if _, err := tx.ExecContext(ctx, `INSERT INTO devices (id, user_id, name, platform, created_at, last_seen_at, last_ip) VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ''))`, deviceID, userID, name, platform, now, now, lastIP); err != nil {
		return PairingResult{}, fmt.Errorf("ghi device: %w", err)
	}

	token, err := newToken()
	if err != nil {
		return PairingResult{}, err
	}
	hash := hashToken(token)
	if _, err := tx.ExecContext(ctx, `INSERT INTO device_tokens (token_hash, device_id, created_at, expires_at, last_used_at) VALUES (?, ?, ?, 0, 0)`, hash, deviceID, now); err != nil {
		return PairingResult{}, fmt.Errorf("ghi token: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE pairing_codes SET consumed_at = ?, device_id = ? WHERE code = ?`, now, deviceID, code); err != nil {
		return PairingResult{}, fmt.Errorf("danh dau pairing code: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return PairingResult{}, err
	}
	return PairingResult{
		Device: Device{ID: deviceID, UserID: userID, Name: name, Platform: platform, CreatedAt: now, LastSeenAt: now, LastIP: lastIP},
		Token:  token,
	}, nil
}

func (s *Service) ListDevices(ctx context.Context, userID string) ([]Device, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, name, COALESCE(platform, ''), created_at, COALESCE(last_seen_at, 0), COALESCE(last_ip, ''), COALESCE(revoked_at, 0) FROM devices WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Device, 0)
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.Platform, &d.CreatedAt, &d.LastSeenAt, &d.LastIP, &d.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Service) RevokeDevice(ctx context.Context, userID, deviceID string) error {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `UPDATE devices SET revoked_at = ? WHERE id = ? AND user_id = ?`, now, deviceID, userID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrInvalidPayload
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM device_tokens WHERE device_id = ?`, deviceID); err != nil {
		return err
	}
	return nil
}

func (s *Service) ResolveToken(ctx context.Context, token string) (Device, error) {
	if !strings.HasPrefix(token, tokenPrefix) {
		return Device{}, ErrInvalidToken
	}
	hash := hashToken(token)
	row := s.db.QueryRowContext(ctx, `SELECT d.id, d.user_id, d.name, COALESCE(d.platform, ''), d.created_at, COALESCE(d.last_seen_at, 0), COALESCE(d.last_ip, ''), COALESCE(d.revoked_at, 0) FROM device_tokens t JOIN devices d ON d.id = t.device_id WHERE t.token_hash = ?`, hash)
	var d Device
	if err := row.Scan(&d.ID, &d.UserID, &d.Name, &d.Platform, &d.CreatedAt, &d.LastSeenAt, &d.LastIP, &d.RevokedAt); err != nil {
		if err == sql.ErrNoRows {
			return Device{}, ErrInvalidToken
		}
		return Device{}, err
	}
	if d.RevokedAt != 0 {
		return Device{}, ErrDeviceRevoked
	}
	now := time.Now().Unix()
	go func() {
		ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = s.db.ExecContext(ctx2, `UPDATE devices SET last_seen_at = ? WHERE id = ?`, now, d.ID)
		_, _ = s.db.ExecContext(ctx2, `UPDATE device_tokens SET last_used_at = ? WHERE token_hash = ?`, now, hash)
	}()
	return d, nil
}

func newPairingCode() (string, error) {
	chunks := make([]string, codeChunkCount)
	buf := make([]byte, codeChunkSize)
	for i := range chunks {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		chunk := make([]byte, codeChunkSize)
		for j, b := range buf {
			chunk[j] = codeAlphabet[int(b)%len(codeAlphabet)]
		}
		chunks[i] = string(chunk)
	}
	return strings.Join(chunks, "-"), nil
}

func normalizeCode(code string) string {
	cleaned := strings.ToUpper(strings.TrimSpace(code))
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	if !strings.Contains(cleaned, "-") && len(cleaned) == codeChunkSize*codeChunkCount {
		cleaned = cleaned[:codeChunkSize] + "-" + cleaned[codeChunkSize:]
	}
	return cleaned
}

func newToken() (string, error) {
	buf := make([]byte, tokenSize)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return tokenPrefix + hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("dev-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}