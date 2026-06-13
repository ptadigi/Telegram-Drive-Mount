package devices

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Service implements the device pairing + token life-cycle for the agent.
type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

const (
	codeTTL        = 5 * time.Minute
	codeAlphabet   = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	codeChunkSize  = 4
	codeChunkCount = 2
	tokenSize      = 32
	tokenPrefix    = "tdd1_"
	maxNameLen     = 64
	maxPlatformLen = 32
)

var (
	ErrPairingNotFound = errors.New("ma ghep thiet bi khong hop le hoac da het han")
	ErrPairingConsumed = errors.New("ma ghep thiet bi da duoc dung")
	ErrPairingExpired  = errors.New("ma ghep thiet bi da het han")
	ErrInvalidToken    = errors.New("token thiet bi khong hop le")
	ErrDeviceRevoked   = errors.New("thiet bi da bi thu hoi")
	ErrInvalidPayload  = errors.New("du lieu thiet bi khong hop le")
)

type Device struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Name       string `json:"name"`
	Platform   string `json:"platform,omitempty"`
	CreatedAt  int64  `json:"created_at"`
	LastSeenAt int64  `json:"last_seen_at"`
	LastIP     string `json:"last_ip,omitempty"`
	RevokedAt  int64  `json:"revoked_at,omitempty"`
}

type PairingCode struct {
	Code      string `json:"code"`
	ExpiresAt int64  `json:"expires_at"`
}

type PairingResult struct {
	Device Device `json:"device"`
	Token  string `json:"token"`
}

// CreateApiToken mints a long-lived API/automation token for a user WITHOUT a
// pairing code (the PWA session is already authenticated). It reuses the
// device + device_tokens tables so the token works exactly like a paired
// device token (Authorization: Device <token>) and can be revoked. The raw
// token is returned once; only its hash is stored.
func (s *Service) CreateApiToken(ctx context.Context, userID, name string) (PairingResult, error) {
	if userID == "" {
		return PairingResult{}, ErrInvalidPayload
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "API token"
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}
	now := time.Now().Unix()
	deviceID := newID()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PairingResult{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO devices (id, user_id, name, platform, created_at, last_seen_at, last_ip) VALUES (?, ?, ?, 'api', ?, ?, NULL)`, deviceID, userID, name, now, now); err != nil {
		return PairingResult{}, fmt.Errorf("ghi device: %w", err)
	}
	token, err := newToken()
	if err != nil {
		return PairingResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO device_tokens (token_hash, device_id, created_at, expires_at, last_used_at) VALUES (?, ?, ?, 0, 0)`, hashToken(token), deviceID, now); err != nil {
		return PairingResult{}, fmt.Errorf("ghi token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return PairingResult{}, err
	}
	return PairingResult{
		Device: Device{ID: deviceID, UserID: userID, Name: name, Platform: "api", CreatedAt: now, LastSeenAt: now},
		Token:  token,
	}, nil
}
