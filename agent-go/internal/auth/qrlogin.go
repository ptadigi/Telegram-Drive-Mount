package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
)

type QRState string

const (
	QRStateIdle             QRState = "idle"
	QRStatePending          QRState = "pending"
	QRStateAwaitingPassword QRState = "awaiting_password"
	QRStateAuthorized       QRState = "authorized"
	QRStateError            QRState = "error"
	QRStateExpired          QRState = "expired"
)

type QRStatus struct {
	State     QRState `json:"state"`
	TokenURL  string  `json:"token_url,omitempty"`
	ExpiresAt int64   `json:"expires_at,omitempty"`
	Error     string  `json:"error,omitempty"`
}

type qrSession struct {
	cancel    context.CancelFunc
	state     QRState
	tokenURL  string
	expiresAt time.Time
	errorMsg  string
	password  chan string
	done      chan struct{}
	mu        sync.Mutex
}

func (s *Service) qrSnapshot() QRStatus {
	s.qrMu.Lock()
	defer s.qrMu.Unlock()
	if s.qr == nil {
		return QRStatus{State: QRStateIdle}
	}
	s.qr.mu.Lock()
	defer s.qr.mu.Unlock()
	status := QRStatus{
		State:    s.qr.state,
		TokenURL: s.qr.tokenURL,
		Error:    s.qr.errorMsg,
	}
	if !s.qr.expiresAt.IsZero() {
		status.ExpiresAt = s.qr.expiresAt.Unix()
	}
	return status
}

func (s *Service) GetQRStatus() QRStatus {
	return s.qrSnapshot()
}

func (s *Service) CancelQR() {
	s.qrMu.Lock()
	session := s.qr
	s.qr = nil
	s.qrMu.Unlock()
	if session == nil {
		return
	}
	session.cancel()
	<-session.done
}

func (s *Service) StartQR(parent context.Context) (QRStatus, error) {
	cfg := s.currentConfig()
	if cfg.Telegram.APIID == 0 || cfg.Telegram.APIHash == "" {
		return QRStatus{}, ErrTelegramConfigMissing
	}
	s.CancelQR()

	ctx, cancel := context.WithCancel(context.Background())
	session := &qrSession{
		cancel:   cancel,
		state:    QRStatePending,
		password: make(chan string, 1),
		done:     make(chan struct{}),
	}
	s.qrMu.Lock()
	s.qr = session
	s.qrMu.Unlock()

	disp := tg.NewUpdateDispatcher()
	gaps := updates.New(updates.Config{Handler: disp})
	storage, _ := newSessionStorage(cfg.Telegram.SessionPath)
	if storage == nil {
		storage = &telegram.FileSessionStorage{Path: cfg.Telegram.SessionPath}
	}
	client := telegram.NewClient(cfg.Telegram.APIID, cfg.Telegram.APIHash, telegram.Options{
		SessionStorage: storage,
		UpdateHandler:  gaps,
	})

	go func() {
		defer close(session.done)
		runErr := client.Run(ctx, func(runCtx context.Context) error {
			qr := client.QR()
			loggedIn := qrlogin.OnLoginToken(disp)
			authResult, qrErr := qr.Auth(runCtx, loggedIn, func(_ context.Context, token qrlogin.Token) error {
				session.mu.Lock()
				session.tokenURL = token.URL()
				session.expiresAt = token.Expires()
				session.state = QRStatePending
				session.mu.Unlock()
				return nil
			})
			if qrErr != nil {
				if isPasswordRequired(qrErr) {
					session.mu.Lock()
					session.state = QRStateAwaitingPassword
					session.mu.Unlock()
					select {
					case password := <-session.password:
						if _, pwErr := client.Auth().Password(runCtx, password); pwErr != nil {
							return pwErr
						}
					case <-runCtx.Done():
						return runCtx.Err()
					}
				} else {
					return qrErr
				}
			} else {
				_ = authResult
			}
			return nil
		})
		session.mu.Lock()
		switch {
		case runErr == nil:
			session.state = QRStateAuthorized
			session.errorMsg = ""
		case errors.Is(runErr, context.Canceled):
			session.state = QRStateIdle
		default:
			if strings.Contains(runErr.Error(), "AUTH_TOKEN_EXPIRED") {
				session.state = QRStateExpired
			} else {
				session.state = QRStateError
			}
			session.errorMsg = runErr.Error()
		}
		session.mu.Unlock()
	}()

	deadline := time.Now().Add(45 * time.Second)
	for {
		status := s.qrSnapshot()
		if status.TokenURL != "" || status.State != QRStatePending || time.Now().After(deadline) {
			return status, nil
		}
		select {
		case <-parent.Done():
			return s.qrSnapshot(), parent.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
}

func (s *Service) SubmitQRPassword(password string) error {
	if password == "" {
		return errors.New("vui lòng nhập mật khẩu xác minh hai bước")
	}
	s.qrMu.Lock()
	session := s.qr
	s.qrMu.Unlock()
	if session == nil {
		return errors.New("chưa có phiên QR đang chờ")
	}
	session.mu.Lock()
	state := session.state
	session.mu.Unlock()
	if state != QRStateAwaitingPassword {
		return fmt.Errorf("phiên QR không yêu cầu mật khẩu (state=%s)", state)
	}
	select {
	case session.password <- password:
		return nil
	default:
		return errors.New("đã nhận mật khẩu, đang xử lý")
	}
}
