package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/database"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/observability"
)

const telemetryServiceName = "rekonsiliasi-kas-api"

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

	stack, err := observability.Setup(ctx, observabilityConfig(cfg))
	if err != nil {
		return fmt.Errorf("setup observability: %w", err)
	}
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := stack.Shutdown(shutdownContext); err != nil {
			fmt.Fprintf(os.Stderr, "telemetry shutdown: %v\n", err)
		}
	}()

	logger := stack.Logger

	var db *database.DB
	if cfg.Database.Enabled {
		db, err = database.New(ctx, cfg.Database.ToDatabaseConfig())
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
		defer db.Close()
	} else {
		logger.WarnContext(ctx, "database is disabled; configure database.enabled and DATABASE_URL before using domain endpoints")
	}

	httpClient := stack.HTTPClient(&http.Client{Timeout: 30 * time.Second})
	svc := service.NewService(cfg, db, httpClient)
	sseManager, err := echohttp.NewSSEManager(cfg, svc, logger)
	if err != nil {
		return fmt.Errorf("create sse manager: %w", err)
	}
	if err := sseManager.Start(ctx); err != nil {
		return fmt.Errorf("start sse manager: %w", err)
	}
	svc.SetStateNotifier(echohttp.NewSSENotifier(sseManager, logger))

	return echohttp.RunEchoHTTPService(ctx, cfg, svc, stack, sseManager)
}

func observabilityConfig(cfg config.Config) observability.Config {
	otelConfig := observability.ConfigFromEnv()
	if strings.TrimSpace(otelConfig.ServiceName) == "" || otelConfig.ServiceName == "unknown-service" {
		otelConfig.ServiceName = telemetryServiceName
	}
	if strings.TrimSpace(otelConfig.ServiceVersion) == "" {
		otelConfig.ServiceVersion = service.Version
	}
	if strings.TrimSpace(otelConfig.Environment) == "" {
		otelConfig.Environment = telemetryEnvironment(cfg.App.Environment)
	}

	return otelConfig
}

func telemetryEnvironment(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "production", "prod":
		return "prod"
	case "staging", "stage":
		return "staging"
	case "development", "dev", "local", "":
		return "dev"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}
