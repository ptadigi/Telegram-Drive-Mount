package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gotd/td/telegram"
	gotdauth "github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"telegram-drive-agent/internal/config"
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
}

type Status struct {
	Configured    bool   `json:"configured"`
	SessionExists bool   `json:"session_exists"`
	LoginStarted  bool   `json:"login_started"`
	Phone         string `json:"phone,omitempty"`
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
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	return Status{
		Configured:    s.cfg.Telegram.APIID != 0 && s.cfg.Telegram.APIHash != "",
		SessionExists: fileExists(s.cfg.Telegram.SessionPath),
		LoginStarted:  s.codeHash != "",
		Phone:         s.phone,
	}
}

func (s *Service) StartLogin(ctx context.Context, input StartLoginInput) (StartLoginResult, error) {
	cfg := s.currentConfig()
	if cfg.Telegram.APIID == 0 || cfg.Telegram.APIHash == "" {
		return StartLoginResult{}, ErrTelegramConfigMissing
	}
	if input.Phone == "" {
		return StartLoginResult{}, errors.New("vui lòng nhập số điện thoại")
	}

	client := newClient(cfg)
	var sent tg.AuthSentCodeClass
	if err := client.Run(ctx, func(runCtx context.Context) error {
		var err error
		sent, err = client.Auth().SendCode(runCtx, input.Phone, gotdauth.SendCodeOptions{})
		return err
	}); err != nil {
		return StartLoginResult{}, fmt.Errorf("gửi mã Telegram: %w", err)
	}

	sentCode, ok := sent.(*tg.AuthSentCode)
	if !ok {
		return StartLoginResult{}, fmt.Errorf("Telegram trả về loại mã không hỗ trợ: %T", sent)
	}

	s.mu.Lock()
	s.phone = input.Phone
	s.codeHash = sentCode.PhoneCodeHash
	s.mu.Unlock()

	return StartLoginResult{NextStep: "code", Phone: input.Phone}, nil
}

func (s *Service) SubmitCode(ctx context.Context, input SubmitCodeInput) (SubmitResult, error) {
	cfg := s.currentConfig()
	phone, codeHash := s.loginState()

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
		return SubmitResult{}, errors.New("vui lòng nhập mật khẩu cloud")
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

func (s *Service) currentConfig() config.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *Service) loginState() (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.phone, s.codeHash
}

func (s *Service) clearLoginState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phone = ""
	s.codeHash = ""
}

func newClient(cfg config.Config) *telegram.Client {
	return telegram.NewClient(cfg.Telegram.APIID, cfg.Telegram.APIHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{Path: cfg.Telegram.SessionPath},
	})
}

func isPasswordRequired(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "SESSION_PASSWORD_NEEDED") || strings.Contains(message, "PASSWORD_NEEDED")
}
