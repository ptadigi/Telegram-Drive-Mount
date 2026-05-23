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

	"telegram-drive-agent/internal/api"
	"telegram-drive-agent/internal/config"
	"telegram-drive-agent/internal/db"
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

	apiServer := api.NewServer(version)

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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
