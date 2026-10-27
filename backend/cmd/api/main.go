package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/database"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	var db *database.DB
	if cfg.Database.Enabled {
		db, err = database.New(ctx, cfg.Database.ToDatabaseConfig())
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
		defer db.Close()
	} else {
		logger.Warn("database is disabled; configure database.enabled and DATABASE_URL before using domain endpoints")
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	svc := service.NewService(cfg, db, httpClient)
	sseManager, err := echohttp.NewSSEManager(cfg, svc, logger)
	if err != nil {
		return fmt.Errorf("create sse manager: %w", err)
	}
	if err := sseManager.Start(ctx); err != nil {
		return fmt.Errorf("start sse manager: %w", err)
	}
	svc.SetStateNotifier(echohttp.NewSSENotifier(sseManager, logger))

	return echohttp.RunEchoHTTPService(ctx, cfg, svc, logger, sseManager)
}

func newLogger(cfg config.Config) *slog.Logger {
	options := &slog.HandlerOptions{Level: slog.LevelInfo}
	if cfg.App.Environment == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, options))
	}

	return slog.New(slog.NewTextHandler(os.Stdout, options))
}
