package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config contains runtime settings for the local desktop agent.
type Config struct {
	Host            string         `json:"host"`
	Port            int            `json:"port"`
	DataDir         string         `json:"data_dir"`
	DatabasePath    string         `json:"database_path"`
	ShutdownTimeout time.Duration  `json:"-"`
	Telegram        TelegramConfig `json:"telegram"`
	Cache           CacheConfig    `json:"cache"`
}

type CacheConfig struct {
	Mode     string `json:"mode"`
	MaxBytes int64  `json:"max_bytes"`
}

type TelegramConfig struct {
	APIID       int    `json:"api_id"`
	APIHash     string `json:"api_hash"`
	SessionPath string `json:"session_path"`
}

// Default returns a safe local-only development configuration.
func Default() Config {
	dataDir := defaultDataDir()
	return Config{
		Host:            "127.0.0.1",
		Port:            8750,
		DataDir:         dataDir,
		DatabasePath:    filepath.Join(dataDir, "metadata.db"),
		ShutdownTimeout: 5 * time.Second,
		Telegram: TelegramConfig{
			SessionPath: filepath.Join(dataDir, "telegram.session"),
		},
		Cache: CacheConfig{
			Mode:     "smart",
			MaxBytes: 5 * 1024 * 1024 * 1024,
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = os.Getenv("TD_AGENT_CONFIG")
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("đọc cấu hình: %w", err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("phân tích cấu hình: %w", err)
		}
	}
	cfg.applyEnv()
	cfg.normalize()
	return cfg, nil
}

func (c *Config) Normalize() {
	c.normalize()
}

func (c *Config) applyEnv() {
	if host := os.Getenv("TD_AGENT_HOST"); host != "" {
		c.Host = host
	}
	if port := os.Getenv("TD_AGENT_PORT"); port != "" {
		if parsed, err := strconv.Atoi(port); err == nil {
			c.Port = parsed
		}
	}
	if dataDir := os.Getenv("TD_AGENT_DATA_DIR"); dataDir != "" {
		c.DataDir = dataDir
	}
}

func (c *Config) normalize() {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 8750
	}
	if c.DataDir == "" {
		c.DataDir = defaultDataDir()
	}
	if c.DatabasePath == "" {
		c.DatabasePath = filepath.Join(c.DataDir, "metadata.db")
	}
	if c.Telegram.SessionPath == "" {
		c.Telegram.SessionPath = filepath.Join(c.DataDir, "telegram.session")
	}
	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = 5 * time.Second
	}
	if c.Cache.Mode == "" {
		c.Cache.Mode = "smart"
	}
	if c.Cache.MaxBytes <= 0 {
		c.Cache.MaxBytes = 5 * 1024 * 1024 * 1024
	}
}

func (c Config) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func defaultDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ".td-agent"
	}
	return filepath.Join(base, "TelegramVirtualDrive", "agent")
}
