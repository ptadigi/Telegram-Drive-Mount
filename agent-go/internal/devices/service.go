package devices

import (
	"database/sql"
	"errors"
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