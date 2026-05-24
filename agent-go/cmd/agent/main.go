package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"telegram-drive-agent/internal/api"
	agentauth "telegram-drive-agent/internal/auth"
	"telegram-drive-agent/internal/config"
	"telegram-drive-agent/internal/db"
	"telegram-drive-agent/internal/drive"
	"telegram-drive-agent/internal/telegramstorage"
	"telegram-drive-agent/internal/tray"
	"telegram-drive-agent/internal/tunnel"
)

const version = "0.1.0-dev"

type driveTunnelListener struct {
	drive *drive.Service
}

func (l driveTunnelListener) OnTunnelStatus(status tunnel.Status) {
	l.drive.SetTunnelStatus(status.Active, status.URL)
}

func main() {
	configPath := flag.String("config", "", "đường dẫn file cấu hình JSON (mặc định lấy từ TD_AGENT_CONFIG)")
	dataDir := flag.String("data-dir", "", "thư mục dữ liệu Agent (ghi đè cấu hình)")
	addr := flag.String("addr", "", "địa chỉ HTTP, ví dụ 0.0.0.0:8750")
	withTray := flag.Bool("tray", false, "chạy kèm tray app desktop")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("không thể tải cấu hình: %v", err)
	}
	if *dataDir != "" {
		cfg.DataDir = *dataDir
		cfg.DatabasePath = ""
		cfg.Telegram.SessionPath = ""
	}
	if *addr != "" {
		host, port := splitAddr(*addr)
		if host != "" {
			cfg.Host = host
		}
		if port > 0 {
			cfg.Port = port
		}
	}
	cfg.Normalize()

	metadataDB, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("không thể mở database metadata: %v", err)
	}
	defer metadataDB.Close()

	authService := agentauth.NewService(cfg)
	telegramStorage := telegramstorage.NewService(cfg)
	driveService := drive.NewService(metadataDB, cfg.DataDir, telegramStorage)
	driveService.SetCachePolicy(cfg.Cache.Mode, cfg.Cache.MaxBytes)
	tunnelSvc := tunnel.New(driveTunnelListener{drive: driveService})
	apiServer := api.NewServer(version, cfg, authService, driveService, tunnelSvc)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var paused atomic.Bool
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if paused.Load() {
					continue
				}
				_, _ = driveService.SyncPendingToTelegram(ctx)
			}
		}
	}()
	go driveService.SyncRootWatcher(ctx)
	go driveService.CacheWorker(ctx, 30*time.Second)
	go driveService.BackupWorker(ctx, 6*time.Hour)

	srv := &http.Server{
		Addr:    cfg.Addr(),
		Handler: apiServer.Handler(),
	}

	go func() {
		log.Printf("telegram-drive-agent listening on http://%s", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	if *withTray {
		execPath, _ := os.Executable()
		go tray.Run(ctx, tray.Hooks{
			BaseURL:  trayBaseURL(cfg),
			DataDir:  cfg.DataDir,
			ExecPath: execPath,
			OnPause: func() {
				paused.Store(true)
				log.Println("đã tạm dừng đồng bộ qua tray")
			},
			OnResume: func() {
				paused.Store(false)
				log.Println("đã bật lại đồng bộ qua tray")
			},
			OnAddRoot: func(path string) error {
				_, err := driveService.CreateSyncRoot(ctx, drive.CreateSyncRootInput{LocalPath: path, Mode: "upload_only"})
				if err != nil {
					log.Printf("không thêm được sync root từ tray: %v", err)
				}
				return err
			},
			OnQuit: func() {
				stop()
			},
		})
	}

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	fmt.Println()
	log.Println("shutting down telegram-drive-agent...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
	log.Println("telegram-drive-agent stopped")
}

func trayBaseURL(cfg config.Config) string {
	host := cfg.Host
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, cfg.Port)
}

func splitAddr(value string) (string, int) {
	host := ""
	port := 0
	for i, ch := range value {
		if ch == ':' {
			host = value[:i]
			fmt.Sscanf(value[i+1:], "%d", &port)
			return host, port
		}
	}
	fmt.Sscanf(value, "%d", &port)
	return host, port
}
