package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-drive-agent/internal/api"
	agentauth "telegram-drive-agent/internal/auth"
	"telegram-drive-agent/internal/config"
	"telegram-drive-agent/internal/db"
	"telegram-drive-agent/internal/drive"
	"telegram-drive-agent/internal/telegramstorage"
)

const version = "0.1.0-dev"

func main() {
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("không thể tải cấu hình: %v", err)
	}

	metadataDB, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("không thể mở database metadata: %v", err)
	}
	defer metadataDB.Close()

	authService := agentauth.NewService(cfg)
	telegramStorage := telegramstorage.NewService(cfg)
	driveService := drive.NewService(metadataDB, cfg.DataDir, telegramStorage)
	apiServer := api.NewServer(version, cfg, authService, driveService)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go driveService.SyncWorker(ctx, 2*time.Second)
	go driveService.SyncRootWatcher(ctx)

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
