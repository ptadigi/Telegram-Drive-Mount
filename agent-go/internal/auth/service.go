package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	gotdauth "github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"telegram-drive-agent/internal/config"
	"telegram-drive-agent/internal/secret"
)

var (
	ErrTelegramConfigMissing = errors.New("chưa cấu hình API Telegram cho Go Agent")
	ErrLoginNotStarted       = errors.New("chưa bắt đầu phiên đăng nhập")
)

type Service struct {
	cfg config.Config

	mu       sync.Mutex
	phone    string
	codeHash string
	codeType string

	qrMu sync.Mutex
	qr   *qrSession
}

type Status struct {
	Configured    bool   `json:"configured"`
	SessionExists bool   `json:"session_exists"`
	LoginStarted  bool   `json:"login_started"`
	Authorized    bool   `json:"authorized"`
	Phone         string `json:"phone,omitempty"`
	CodeType      string `json:"code_type,omitempty"`
}

type StartLoginInput struct {
	Phone string `json:"phone"`
}

type SubmitCodeInput struct {
	Code string `json:"code"`
}

type SubmitPasswordInput struct {
	Password string `json:"password"`
}

type StartLoginResult struct {
	NextStep string `json:"next_step"`
	Phone    string `json:"phone"`
	CodeType string `json:"code_type"`
	Timeout  int    `json:"timeout_sec"`
}

type SubmitResult struct {
	Success  bool   `json:"success"`
	NextStep string `json:"next_step,omitempty"`
}

func NewService(cfg config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) UpdateTelegramConfig(apiID int, apiHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Telegram.APIID = apiID
	s.cfg.Telegram.APIHash = apiHash
	s.phone = ""
	s.codeHash = ""
	s.codeType = ""
}

func (s *Service) Status(ctx context.Context) Status {
	cfg := s.currentConfig()
	phone, codeHash, codeType := s.loginState()
	status := Status{
		Configured:    cfg.Telegram.APIID != 0 && cfg.Telegram.APIHash != "",
		SessionExists: fileExists(cfg.Telegram.SessionPath),
		LoginStarted:  codeHash != "",
		Phone:         phone,
		CodeType:      codeType,
	}
	if status.Configured && status.SessionExists {
		status.Authorized = s.checkAuthorized(ctx, cfg)
	}
	return status
}

func (s *Service) ResetLogin() error {
	s.clearLoginState()
	cfg := s.currentConfig()
	if cfg.Telegram.SessionPath != "" {
		_ = os.Remove(cfg.Telegram.SessionPath)
	}
	return nil
}

func (s *Service) StartLogin(ctx context.Context, input StartLoginInput) (StartLoginResult, error) {
	cfg := s.currentConfig()
	if cfg.Telegram.APIID == 0 || cfg.Telegram.APIHash == "" {
		return StartLoginResult{}, ErrTelegramConfigMissing
	}
	if input.Phone == "" {
		return StartLoginResult{}, errors.New("vui lòng nhập số điện thoại")
	}

	s.clearLoginState()
	client := newClient(cfg)
	var sent tg.AuthSentCodeClass
	if err := client.Run(ctx, func(runCtx context.Context) error {
		var err error
		sent, err = client.Auth().SendCode(runCtx, input.Phone, gotdauth.SendCodeOptions{})
		return err
	}); err != nil {
		if isAuthRestart(err) {
			s.clearLoginState()
			return StartLoginResult{}, errors.New("Telegram yêu cầu khởi động lại phiên đăng nhập. Vui lòng bấm gửi mã lại sau vài giây")
		}
		return StartLoginResult{}, fmt.Errorf("gửi mã Telegram: %w", err)
	}

	sentCode, ok := sent.(*tg.AuthSentCode)
	if !ok {
		return StartLoginResult{}, fmt.Errorf("Telegram trả về loại mã không hỗ trợ: %T", sent)
	}

	codeType := sentCode.Type.TypeName()
	s.mu.Lock()
	s.phone = input.Phone
	s.codeHash = sentCode.PhoneCodeHash
	s.codeType = codeType
	s.mu.Unlock()

	timeout, _ := sentCode.GetTimeout()
	return StartLoginResult{NextStep: "code", Phone: input.Phone, CodeType: codeType, Timeout: timeout}, nil
}

func (s *Service) SubmitCode(ctx context.Context, input SubmitCodeInput) (SubmitResult, error) {
	cfg := s.currentConfig()
	phone, codeHash, _ := s.loginState()

	if phone == "" || codeHash == "" {
		return SubmitResult{}, ErrLoginNotStarted
	}
	if input.Code == "" {
		return SubmitResult{}, errors.New("vui lòng nhập mã Telegram")
	}

	client := newClient(cfg)
	if err := client.Run(ctx, func(runCtx context.Context) error {
		_, err := client.Auth().SignIn(runCtx, phone, input.Code, codeHash)
		return err
	}); err != nil {
		if isPasswordRequired(err) {
			return SubmitResult{Success: false, NextStep: "password"}, nil
		}
		return SubmitResult{}, fmt.Errorf("xác minh mã Telegram: %w", err)
	}

	s.clearLoginState()
	return SubmitResult{Success: true}, nil
}

func (s *Service) SubmitPassword(ctx context.Context, input SubmitPasswordInput) (SubmitResult, error) {
	cfg := s.currentConfig()
	if input.Password == "" {
		return SubmitResult{}, errors.New("vui lòng nhập mật khẩu xác minh hai bước")
	}

	client := newClient(cfg)
	if err := client.Run(ctx, func(runCtx context.Context) error {
		_, err := client.Auth().Password(runCtx, input.Password)
		return err
	}); err != nil {
		return SubmitResult{}, fmt.Errorf("xác minh mật khẩu Telegram: %w", err)
	}

	s.clearLoginState()
	return SubmitResult{Success: true}, nil
}

func (s *Service) checkAuthorized(ctx context.Context, cfg config.Config) bool {
	client := newClient(cfg)
	ok := false
	_ = client.Run(ctx, func(runCtx context.Context) error {
		status, err := client.Auth().Status(runCtx)
		if err != nil {
			return nil
		}
		ok = status.Authorized
		return nil
	})
	return ok
}

func (s *Service) currentConfig() config.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *Service) loginState() (string, string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.phone, s.codeHash, s.codeType
}

func (s *Service) clearLoginState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phone = ""
	s.codeHash = ""
	s.codeType = ""
}

func newClient(cfg config.Config) *telegram.Client {
	storage, err := newSessionStorage(cfg.Telegram.SessionPath)
	if err != nil {
		// fall back to plain file storage so login flow still works in dev
		storage = &telegram.FileSessionStorage{Path: cfg.Telegram.SessionPath}
	}
	return telegram.NewClient(cfg.Telegram.APIID, cfg.Telegram.APIHash, telegram.Options{
		SessionStorage: storage,
	})
}

func newSessionStorage(path string) (session.Storage, error) {
	key, err := secret.LoadKey()
	if err != nil {
		return nil, err
	}
	if key == nil {
		return &telegram.FileSessionStorage{Path: path}, nil
	}
	return &secret.EncryptedSessionStorage{Path: path, Key: key}, nil
}

func isPasswordRequired(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "SESSION_PASSWORD_NEEDED") ||
		strings.Contains(message, "PASSWORD_NEEDED")
}

func isAuthRestart(err error) bool {
	return err != nil && strings.Contains(err.Error(), "AUTH_RESTART")
}
