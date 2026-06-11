package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// NormalizeURL trims and ensures a scheme. Defaults to https unless loopback.
func NormalizeURL(raw string) (string, error) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", errors.New("thiếu URL máy chủ")
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		if strings.HasPrefix(u, "127.0.0.1") || strings.HasPrefix(u, "localhost") {
			u = "http://" + u
		} else {
			u = "https://" + u
		}
	}
	return strings.TrimRight(u, "/"), nil
}

// ServerInfo is the result of probing a server URL.
type ServerInfo struct {
	OK      bool   `json:"ok"`
	URL     string `json:"url"`
	Service string `json:"service,omitempty"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}

// TestServer checks that the URL hosts a Telegram Drive agent by calling /health.
func TestServer(ctx context.Context, rawURL string) ServerInfo {
	url, err := NormalizeURL(rawURL)
	if err != nil {
		return ServerInfo{Error: err.Error()}
	}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url+"/health", nil)
	if err != nil {
		return ServerInfo{URL: url, Error: err.Error()}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ServerInfo{URL: url, Error: "không kết nối được máy chủ: " + err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ServerInfo{URL: url, Error: fmt.Sprintf("máy chủ trả về mã %d", resp.StatusCode)}
	}
	var health struct {
		OK      bool   `json:"ok"`
		Service string `json:"service"`
		Version string `json:"version"`
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(raw, &health)
	if health.Service == "" {
		return ServerInfo{URL: url, Error: "endpoint không phải Telegram Drive agent"}
	}
	return ServerInfo{OK: true, URL: url, Service: health.Service, Version: health.Version}
}

// PairResult holds the outcome of exchanging a pairing code for a token.
type PairResult struct {
	BaseURL  string
	Token    string
	DeviceID string
	UserID   string
}

// ExchangePairing posts the pairing code to the server's exchange endpoint.
func ExchangePairing(ctx context.Context, rawURL, code, name string) (PairResult, error) {
	url, err := NormalizeURL(rawURL)
	if err != nil {
		return PairResult{}, err
	}
	if strings.TrimSpace(code) == "" {
		return PairResult{}, errors.New("thiếu mã ghép thiết bị")
	}
	if name == "" {
		host, _ := os.Hostname()
		if host == "" {
			host = "td-client"
		}
		name = host
	}
	body, _ := json.Marshal(map[string]string{
		"code":     strings.TrimSpace(code),
		"name":     name,
		"platform": runtime.GOOS,
	})
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url+"/v1/devices/pair/exchange", bytes.NewReader(body))
	if err != nil {
		return PairResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return PairResult{}, fmt.Errorf("gọi máy chủ: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errResp)
		if errResp.Error != "" {
			return PairResult{}, fmt.Errorf("máy chủ từ chối: %s", errResp.Error)
		}
		return PairResult{}, fmt.Errorf("máy chủ từ chối (HTTP %d)", resp.StatusCode)
	}
	var ok struct {
		Device struct {
			ID     string `json:"id"`
			UserID string `json:"user_id"`
		} `json:"device"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &ok); err != nil {
		return PairResult{}, fmt.Errorf("đọc phản hồi máy chủ: %w", err)
	}
	if ok.Token == "" {
		return PairResult{}, errors.New("máy chủ không trả về token")
	}
	return PairResult{BaseURL: url, Token: ok.Token, DeviceID: ok.Device.ID, UserID: ok.Device.UserID}, nil
}
