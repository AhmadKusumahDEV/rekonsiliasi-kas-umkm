package echohttp

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

const contextKeyRequestStartedAt = "request_started_at"

func requestLogger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			started := time.Now()
			c.Set(contextKeyRequestStartedAt, started)
			err := next(c)
			// Error responses are logged by the central HTTP error handler, where
			// the final status code and user-facing mapping are already known.
			if err != nil {
				return err
			}

			latency := time.Since(started)
			status := c.Response().Status
			if status == 0 {
				status = 200
			}

			attributes := []any{
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", status,
				"latency_ms", latency.Milliseconds(),
				"request_id", requestID(c),
				"remote_ip", c.RealIP(),
			}
			logger.Info("http request", attributes...)
			return nil
		}
	}
}
