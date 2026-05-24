package drive

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ShareConfig struct {
	Mode          string `json:"mode"`
	Domain        string `json:"domain,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
	LocalBaseURL  string `json:"local_base_url"`
	Port          int    `json:"port"`
	GatewayToken  string `json:"-"`
	HealthOK      bool   `json:"health_ok"`
	HealthMessage string `json:"health_message,omitempty"`
	UpdatedAt     int64  `json:"updated_at,omitempty"`
}

func (s *Service) GetShareConfig(ctx context.Context) (ShareConfig, error) {
	row := s.db.QueryRowContext(ctx, `SELECT mode, COALESCE(domain, ''), COALESCE(base_url, ''), COALESCE(local_base_url, ''), COALESCE(port, 8750), COALESCE(gateway_token, ''), COALESCE(health_ok, 0), COALESCE(health_message, ''), COALESCE(updated_at, 0) FROM share_config WHERE id = 'default'`)
	var cfg ShareConfig
	var ok int
	if err := row.Scan(&cfg.Mode, &cfg.Domain, &cfg.BaseURL, &cfg.LocalBaseURL, &cfg.Port, &cfg.GatewayToken, &ok, &cfg.HealthMessage, &cfg.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ShareConfig{Mode: "lan", LocalBaseURL: "", Port: 8750}, nil
		}
		return ShareConfig{}, err
	}
	cfg.HealthOK = ok == 1
	return cfg, nil
}

type UpdateShareConfigInput struct {
	Mode    string `json:"mode"`
	Domain  string `json:"domain"`
	BaseURL string `json:"base_url"`
}

func (s *Service) UpdateShareConfig(ctx context.Context, input UpdateShareConfigInput, localBaseURL string) (ShareConfig, error) {
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = "lan"
	}
	if mode != "lan" && mode != "domain" && mode != "tunnel" {
		return ShareConfig{}, fmt.Errorf("chế độ chia sẻ không hợp lệ")
	}
	domain := strings.TrimSpace(input.Domain)
	baseURL := strings.TrimSpace(input.BaseURL)
	if mode == "domain" {
		if domain == "" {
			return ShareConfig{}, fmt.Errorf("vui lòng nhập tên miền chia sẻ")
		}
		if baseURL == "" {
			baseURL = "https://" + domain
		}
	}
	now := time.Now().Unix()
	cfg, err := s.GetShareConfig(ctx)
	if err != nil {
		return ShareConfig{}, err
	}
	cfg.Mode = mode
	cfg.Domain = domain
	cfg.BaseURL = baseURL
	cfg.LocalBaseURL = localBaseURL
	cfg.UpdatedAt = now
	if cfg.GatewayToken == "" {
		token, err := generateSlug()
		if err != nil {
			return ShareConfig{}, err
		}
		cfg.GatewayToken = token
	}
	cfg.HealthOK, cfg.HealthMessage = s.checkShareHealth(cfg)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO share_config (id, mode, domain, base_url, local_base_url, port, gateway_token, health_ok, health_message, updated_at)
		VALUES ('default', ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET mode = excluded.mode, domain = excluded.domain, base_url = excluded.base_url, local_base_url = excluded.local_base_url, port = excluded.port, gateway_token = excluded.gateway_token, health_ok = excluded.health_ok, health_message = excluded.health_message, updated_at = excluded.updated_at
	`, cfg.Mode, cfg.Domain, cfg.BaseURL, cfg.LocalBaseURL, cfg.Port, cfg.GatewayToken, boolToInt(cfg.HealthOK), cfg.HealthMessage, now)
	if err != nil {
		return ShareConfig{}, fmt.Errorf("ghi cấu hình chia sẻ: %w", err)
	}
	s.events.Publish("share.config", cfg)
	return cfg, nil
}

func (s *Service) ShareLink(cfg ShareConfig, slug string) string {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = strings.TrimRight(cfg.LocalBaseURL, "/")
	}
	if base == "" {
		base = "http://127.0.0.1"
	}
	return base + "/share/" + slug
}

func (s *Service) checkShareHealth(cfg ShareConfig) (bool, string) {
	if cfg.Mode == "lan" {
		return true, "Đang dùng chế độ LAN, link chỉ mở trong mạng nội bộ"
	}
	if cfg.Mode == "tunnel" {
		return true, "Đã bật Cloudflare Tunnel ở chế độ thử nghiệm"
	}
	if cfg.BaseURL == "" {
		return false, "Chưa có base URL để kiểm tra"
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(strings.TrimRight(cfg.BaseURL, "/") + "/.td-check")
	if err != nil {
		return false, fmt.Sprintf("Không kết nối được %s: %v", cfg.BaseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("Domain trả về mã %d", resp.StatusCode)
	}
	return true, "Đã kiểm tra domain thành công"
}
