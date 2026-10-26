// Package echohttp is the HTTP transport layer. Handlers stay thin: they bind,
// validate, read identity from the request context, call a service method, and
// write the response. SQL and transactions never live here.
package echohttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// NewEcho builds the HTTP engine. It is exported so handler and route tests can
// exercise the real middleware and error mapping without opening a socket.
func NewEcho(cfg config.Config, s *service.Service, logger *slog.Logger) *echo.Echo {
	if logger == nil {
		logger = slog.Default()
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = handleEchoError(logger)
	e.Validator = NewCustomValidator()

	e.Use(middleware.Recover())
	e.Use(securityHeaders)
	e.Use(corsMiddleware(cfg.Server.AllowedOrigins))
	e.Use(middleware.BodyLimit("1M"))
	e.Use(middleware.RequestID())
	e.Use(requestLogger(logger))

	e.GET("/health/live", liveness(s))
	e.GET("/health/ready", readiness(s))

	// Domain routes are registered here. Keep the group for authenticated
	// endpoints once the authentication middleware exists:
	//   v1 := e.Group("/v1", authMiddleware)
	//   v1.GET("/contoh", listContoh(s))

	return e
}

func RunEchoHTTPService(ctx context.Context, cfg config.Config, s *service.Service, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	e := NewEcho(cfg, s, logger)
	e.Server.ReadTimeout = cfg.Server.ReadTimeout
	e.Server.WriteTimeout = cfg.Server.WriteTimeout
	e.Server.IdleTimeout = cfg.Server.IdleTimeout

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server listening",
			"address", cfg.Server.Address(),
			"service", cfg.App.Name,
			"version", service.Version,
		)
		serverErrors <- e.Start(cfg.Server.Address())
	}()

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		logger.Info("stopping http server")
		if err := e.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}

		return nil
	case err := <-serverErrors:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("run http server: %w", err)
	}
}
