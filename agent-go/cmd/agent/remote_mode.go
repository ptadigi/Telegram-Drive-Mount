package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"telegram-drive-agent/internal/remote"
	"telegram-drive-agent/internal/vfs"
)

// runPair runs the interactive pairing flow when the user invokes
// `td-agent --pair`. It asks for the VPS base URL + a code shown in PWA,
// posts to /v1/devices/pair/exchange, and saves the resulting token.
func runPair(baseURL, code, name, tokenPath string) error {
	reader := bufio.NewReader(os.Stdin)
	if baseURL == "" {
		fmt.Print("VPS URL (https://drive.example.com): ")
		raw, _ := reader.ReadString('\n')
		baseURL = strings.TrimSpace(raw)
	}
	if baseURL == "" {
		return errors.New("thieu VPS URL")
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	if strings.HasPrefix(baseURL, "http://") && os.Getenv("TD_AGENT_INSECURE") != "1" {
		return errors.New("yeu cau HTTPS, de bypass dat TD_AGENT_INSECURE=1 (dev only)")
	}
	if code == "" {
		fmt.Print("Pairing code (vd 4F2A-9K2X): ")
		raw, _ := reader.ReadString('\n')
		code = strings.TrimSpace(raw)
	}
	if code == "" {
		return errors.New("thieu pairing code")
	}
	if name == "" {
		host, _ := os.Hostname()
		if host == "" {
			host = "td-client"
		}
		name = host
	}

	body, _ := json.Marshal(map[string]string{
		"code":     code,
		"name":     name,
		"platform": runtime.GOOS,
	})
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/devices/pair/exchange"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("goi VPS: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var errResp struct{ Error string }
		_ = json.Unmarshal(raw, &errResp)
		if errResp.Error != "" {
			return fmt.Errorf("VPS tu choi: %s", errResp.Error)
		}
		return fmt.Errorf("VPS tu choi (HTTP %d)", resp.StatusCode)
	}
	var ok struct {
		Device struct {
			ID     string `json:"id"`
			UserID string `json:"user_id"`
		} `json:"device"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &ok); err != nil {
		return fmt.Errorf("parse VPS response: %w", err)
	}
	if ok.Token == "" {
		return errors.New("VPS khong tra ve token")
	}
	tok := remote.Token{
		BaseURL:  baseURL,
		Token:    ok.Token,
		DeviceID: ok.Device.ID,
		UserID:   ok.Device.UserID,
	}
	if tokenPath == "" {
		tokenPath = remote.DefaultTokenPath()
	}
	if err := remote.SaveToken(tokenPath, tok); err != nil {
		return err
	}
	fmt.Printf("Da pair voi VPS %s. Token luu tai %s\n", parseHost(baseURL), tokenPath)
	return nil
}

// runRemoteClient starts the agent in thin-client mode: no SQLite, no Telegram
// session, no PWA serving. Just mount FUSE backed by remote.Backend.
func runRemoteClient(ctx context.Context, baseURL, tokenPath, mountPoint string) error {
	if tokenPath == "" {
		tokenPath = remote.DefaultTokenPath()
	}
	tok, err := remote.LoadToken(tokenPath)
	if err != nil {
		return fmt.Errorf("doc token (%s): %w", tokenPath, err)
	}
	if baseURL == "" {
		baseURL = tok.BaseURL
	}
	if baseURL == "" {
		return errors.New("thieu VPS URL")
	}
	backend := remote.NewBackend(baseURL, tok.Token, 60*time.Second)
	manager := vfs.NewManagerWithBackend(backend, os.TempDir())
	status, err := manager.Mount(ctx, mountPoint)
	if err != nil {
		return fmt.Errorf("mount: %w", err)
	}
	log.Printf("td-agent --remote: mounted at %s (backend=%s)", status.MountPoint, status.Backend)
	<-ctx.Done()
	manager.Shutdown()
	return nil
}

func parseHost(rawurl string) string {
	u, err := url.Parse(rawurl)
	if err != nil {
		return rawurl
	}
	return u.Host
}