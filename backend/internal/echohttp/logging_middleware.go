package echohttp

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/observability"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/trace"
)

const contextKeyRequestStartedAt = "request_started_at"

func requestLogger(logger *slog.Logger, stack *observability.Stack) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			started := time.Now()
			c.Set(contextKeyRequestStartedAt, started)
			err := next(c)

			latency := time.Since(started)
			status := c.Response().Status
			if status == 0 {
				status = http.StatusOK
			}
			route := c.Path()
			ctx := c.Request().Context()
			if stack != nil {
				stack.LogRequest(ctx, c.Request().Method, route, status, latency)
				return err
			}

			attributes := []any{
				"http.method", c.Request().Method,
				"http.route", route,
				"http.status", status,
				"duration_ms", float64(latency.Microseconds()) / 1000,
			}
			if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
				attributes = append(attributes,
					"trace_id", spanContext.TraceID().String(),
					"span_id", spanContext.SpanID().String(),
				)
			}
			logger.InfoContext(ctx, "http request", attributes...)
			return err
		}
	}
}
