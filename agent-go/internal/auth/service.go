package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gotd/td/telegram"
	gotdauth "github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"telegram-drive-agent/internal/config"
)

var (
	ErrTelegramConfigMissing = errors.New("chưa cấu hình API ID hoặc API Hash Telegram")
	ErrLoginNotStarted       = errors.New("chưa bắt đầu phiên đăng nhập")
)

type Service struct {
	cfg config.Config

	mu        sync.Mutex
	phone     string
	codeHash  string
	client    *telegram.Client
	cancelRun context.CancelFunc
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
	s.client = nil
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
	if s.cfg.Telegram.APIID == 0 || s.cfg.Telegram.APIHash == "" {
		return StartLoginResult{}, ErrTelegramConfigMissing
	}
	if input.Phone == "" {
		return StartLoginResult{}, errors.New("vui lòng nhập số điện thoại")
	}

	client, err := s.ensureClient()
	if err != nil {
		return StartLoginResult{}, err
	}

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
	s.mu.Lock()
	phone := s.phone
	codeHash := s.codeHash
	s.mu.Unlock()

	if phone == "" || codeHash == "" {
		return SubmitResult{}, ErrLoginNotStarted
	}
	if input.Code == "" {
		return SubmitResult{}, errors.New("vui lòng nhập mã Telegram")
	}

	client, err := s.ensureClient()
	if err != nil {
		return SubmitResult{}, err
	}

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
	if input.Password == "" {
		return SubmitResult{}, errors.New("vui lòng nhập mật khẩu cloud")
	}

	client, err := s.ensureClient()
	if err != nil {
		return SubmitResult{}, err
	}

	if err := client.Run(ctx, func(runCtx context.Context) error {
		_, err := client.Auth().Password(runCtx, input.Password)
		return err
	}); err != nil {
		return SubmitResult{}, fmt.Errorf("xác minh mật khẩu Telegram: %w", err)
	}

	s.clearLoginState()
	return SubmitResult{Success: true}, nil
}

func (s *Service) ensureClient() (*telegram.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return s.client, nil
	}

	s.client = telegram.NewClient(s.cfg.Telegram.APIID, s.cfg.Telegram.APIHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{Path: s.cfg.Telegram.SessionPath},
	})
	return s.client, nil
}

func (s *Service) clearLoginState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phone = ""
	s.codeHash = ""
}

func isPasswordRequired(err error) bool {
	return err != nil && (contains(err.Error(), "SESSION_PASSWORD_NEEDED") || contains(err.Error(), "PASSWORD_NEEDED"))
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && index(s, substr) >= 0)
}

func index(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
