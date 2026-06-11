package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"telegram-drive-agent/internal/api"
	agentauth "telegram-drive-agent/internal/auth"
	"telegram-drive-agent/internal/config"
	"telegram-drive-agent/internal/db"
	"telegram-drive-agent/internal/desktop"
	"telegram-drive-agent/internal/devices"
	"telegram-drive-agent/internal/drive"
	"telegram-drive-agent/internal/telegramstorage"
	"telegram-drive-agent/internal/remote"
	"telegram-drive-agent/internal/tray"
	"telegram-drive-agent/internal/tunnel"
	"telegram-drive-agent/internal/users"
	"telegram-drive-agent/internal/vfs"
)

const version = "1.7.3"

var errSetupWindowUnavailable = errors.New("cửa sổ thiết lập không khả dụng trên bản build này")


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
	pairMode := flag.Bool("pair", false, "ghép thiết bị với VPS rồi thoát")
	pairBase := flag.String("pair-url", "", "URL VPS khi --pair (vd https://drive.example.com)")
	pairCode := flag.String("pair-code", "", "pairing code lấy trên PWA (vd 4F2A-9K2X)")
	pairName := flag.String("pair-name", "", "tên thiết bị (mặc định hostname)")
	remoteMode := flag.Bool("remote", false, "thin-client mode: bỏ SQLite/Telegram, chỉ mount qua VPS")
	remoteBase := flag.String("remote-url", "", "VPS URL khi --remote (mặc định lấy từ token đã lưu)")
	remoteMount := flag.String("remote-mount", "", "điểm mount khi --remote (mặc định T: hoặc /Volumes/...)")
	tokenPath := flag.String("token-file", "", "đường dẫn token file (mặc định theo $XDG_CONFIG_HOME)")
	mountOnStart := flag.Bool("mount-on-start", false, "tự mount ổ ảo ngay khi agent khởi động")
	mountPoint := flag.String("mount-point", "", "điểm mount khi --mount-on-start (mặc định T: / /Volumes/...)")
	setupWindow := flag.String("setup-window", "", "(nội bộ) mở cửa sổ thiết lập WebView2 tới URL")
	flag.Parse()

	if *setupWindow != "" {
		if err := runSetupWindow(*setupWindow); err != nil {
			_ = openBrowser(*setupWindow)
		}
		return
	}

	if *pairMode {
		if err := runPair(*pairBase, *pairCode, *pairName, *tokenPath); err != nil {
			log.Fatalf("pair: %v", err)
		}
		return
	}

	if *remoteMode {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := runRemoteClient(ctx, *remoteBase, *tokenPath, *remoteMount); err != nil {
			log.Fatalf("remote: %v", err)
		}
		return
	}

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

	// Desktop builds ship the PWA next to the executable. When pwa_dir isn't
	// configured, auto-detect <exeDir>/pwa so the agent serves the UI and the
	// /setup onboarding page without any manual config (fixes 404 on /setup).
	if strings.TrimSpace(cfg.PWADir) == "" {
		if exe, err := os.Executable(); err == nil {
			candidate := filepath.Join(filepath.Dir(exe), "pwa")
			if info, statErr := os.Stat(filepath.Join(candidate, "index.html")); statErr == nil && !info.IsDir() {
				cfg.PWADir = candidate
			}
		}
	}

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
	userService := users.New(metadataDB)
	deviceService := devices.New(metadataDB)
	// Choose the mount backend based on desktop onboarding state. In remote
	// mode the virtual drive must stream from the paired server (not the empty
	// local DB), so wire a remote.Backend using the saved device token.
	var mountManager *vfs.Manager
	if *withTray {
		st := desktop.NewStore(cfg.DataDir).Load()
		if desktop.Mode(st.Mode) == desktop.ModeRemote {
			if tok, tokErr := remote.LoadToken(remote.DefaultTokenPath()); tokErr == nil {
				backend := remote.NewBackend(tok.BaseURL, tok.Token, 60*time.Second)
				mountManager = vfs.NewManagerWithBackend(backend, cfg.DataDir)
				log.Printf("mount backend: remote (%s)", tok.BaseURL)
			} else {
				log.Printf("chế độ remote nhưng chưa có token (%v), tạm dùng backend local", tokErr)
			}
		}
	}
	if mountManager == nil {
		mountManager = vfs.NewManager(driveService, cfg.DataDir)
	}
	apiServer := api.NewServer(version, cfg, authService, driveService, tunnelSvc, userService, mountManager, deviceService)
	apiServer.SetDesktopMode(*withTray)

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

	if *mountOnStart {
		go func() {
			// Give the HTTP server a beat to come up, then mount.
			time.Sleep(1 * time.Second)
			status, err := mountManager.Mount(ctx, *mountPoint)
			if err != nil {
				log.Printf("mount-on-start lỗi: %v", err)
				return
			}
			log.Printf("đã tự mount ổ ảo tại %s (backend=%s)", status.MountPoint, status.Backend)
		}()
	}

	if *withTray {
		execPath, _ := os.Executable()
		// Honor saved desktop onboarding state: open setup on first run,
		// auto-mount when already configured.
		go func() {
			time.Sleep(1 * time.Second)
			store := desktop.NewStore(cfg.DataDir)
			st := store.Load()
			base := trayBaseURL(cfg)
			switch desktop.Mode(st.Mode) {
			case desktop.ModeLocal:
				if _, err := mountManager.Mount(ctx, st.MountPoint); err != nil {
					log.Printf("tự mount (local) lỗi: %v", err)
				}
			case desktop.ModeRemote:
				if _, err := mountManager.Mount(ctx, st.MountPoint); err != nil {
					log.Printf("tự mount (remote) lỗi: %v", err)
				}
			default:
				openSetup(execPath, base+"/setup")
			}
		}()
		go tray.Run(ctx, tray.Hooks{
			DataDir:  cfg.DataDir,
			ExecPath: execPath,
			OnOpenUI: func() {
				_ = openBrowser(resolveUIBaseURL(cfg))
			},
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
			OnMount: func() (string, error) {
				status, err := mountManager.Mount(ctx, "")
				return status.MountPoint, err
			},
			OnUnmount: func() error {
				_, err := mountManager.Unmount()
				return err
			},
			OnSetup: func() {
				openSetup(execPath, trayBaseURL(cfg)+"/setup")
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
		log.Printf("graceful HTTP shutdown failed: %v", err)
	}
	mountManager.Shutdown()
	log.Println("telegram-drive-agent stopped")
}

func trayBaseURL(cfg config.Config) string {
	host := cfg.Host
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, cfg.Port)
}

// resolveUIBaseURL returns the URL the desktop tray should open for the main
// UI. In remote mode the real Drive UI lives on the server the user paired
// with, so we open that; otherwise (local / not configured) we open the
// local agent. Reads fresh state each call so it reflects onboarding changes.
func resolveUIBaseURL(cfg config.Config) string {
	st := desktop.NewStore(cfg.DataDir).Load()
	if desktop.Mode(st.Mode) == desktop.ModeRemote && strings.TrimSpace(st.ServerURL) != "" {
		return strings.TrimRight(st.ServerURL, "/")
	}
	return trayBaseURL(cfg)
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// openSetup opens the onboarding page in a native WebView2 window (spawned as
// a child process so it owns its own UI thread). Falls back to the system
// browser if the window cannot be launched.
func openSetup(execPath, url string) {
	if execPath != "" && runtime.GOOS == "windows" {
		if err := exec.Command(execPath, "--setup-window", url).Start(); err == nil {
			return
		}
	}
	_ = openBrowser(url)
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
