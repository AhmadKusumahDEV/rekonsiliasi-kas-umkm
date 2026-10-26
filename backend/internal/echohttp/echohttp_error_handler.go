package echohttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

const statusClientClosedRequest = 499

type apiErrorResponse struct {
	Code      int    `json:"code"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func handleEchoError(logger *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		statusCode, message := mapHTTPError(err)
		if logger != nil {
			attributes := []any{
				"error", err,
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"request_id", requestID(c),
				"status", statusCode,
			}
			if started, ok := c.Get(contextKeyRequestStartedAt).(time.Time); ok {
				attributes = append(attributes, "latency_ms", time.Since(started).Milliseconds())
			}
			if statusCode >= http.StatusInternalServerError {
				logger.Error("request failed", attributes...)
			} else {
				logger.Warn("request failed", attributes...)
			}
		}

		_ = c.JSON(statusCode, apiErrorResponse{
			Code:      statusCode,
			Status:    statusText(statusCode),
			Message:   message,
			RequestID: requestID(c),
		})
	}
}

func statusText(statusCode int) string {
	if statusCode == statusClientClosedRequest {
		return "client closed request"
	}

	return strings.ToLower(http.StatusText(statusCode))
}

func mapHTTPError(err error) (int, string) {
	var domainError *pkgErr.DomainError
	if errors.As(err, &domainError) {
		statusCode, _ := mapHTTPError(domainError.Kind)
		return statusCode, domainError.Message
	}

	if errors.Is(err, context.Canceled) {
		return statusClientClosedRequest, "Permintaan dibatalkan."
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, "Permintaan melewati batas waktu."
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		return http.StatusBadRequest, validationMessage(validationErrors)
	}

	switch {
	case errors.Is(err, pkgErr.ErrInvalidRequest):
		return http.StatusBadRequest, pkgErr.MsgInvalidRequest
	case errors.Is(err, pkgErr.ErrUnauthorized):
		return http.StatusUnauthorized, pkgErr.MsgUnauthorized
	case errors.Is(err, pkgErr.ErrForbidden):
		return http.StatusForbidden, pkgErr.MsgForbidden
	case errors.Is(err, pkgErr.ErrNotFound):
		return http.StatusNotFound, pkgErr.MsgNotFound
	case errors.Is(err, pkgErr.ErrConflict):
		return http.StatusConflict, pkgErr.MsgConflict
	case errors.Is(err, pkgErr.ErrServiceUnavailable):
		return http.StatusServiceUnavailable, pkgErr.MsgServiceUnavailable
	case errors.Is(err, pkgErr.ErrFailInitializeTx), errors.Is(err, pkgErr.ErrFailCommitTx):
		return http.StatusInternalServerError, pkgErr.MsgInternal
	}

	var echoError *echo.HTTPError
	if errors.As(err, &echoError) {
		switch echoError.Code {
		case http.StatusBadRequest:
			return echoError.Code, pkgErr.MsgInvalidRequest
		case http.StatusNotFound:
			return echoError.Code, "Endpoint yang diminta tidak ditemukan."
		case http.StatusMethodNotAllowed:
			return echoError.Code, "Method HTTP tidak diizinkan untuk endpoint ini."
		case http.StatusRequestEntityTooLarge:
			return echoError.Code, "Ukuran permintaan terlalu besar."
		case http.StatusUnsupportedMediaType:
			return echoError.Code, "Tipe konten permintaan tidak didukung."
		default:
			if echoError.Code < http.StatusInternalServerError {
				return echoError.Code, http.StatusText(echoError.Code)
			}
		}
	}

	return http.StatusInternalServerError, pkgErr.MsgInternal
}

func validationMessage(errors validator.ValidationErrors) string {
	details := make([]string, 0, 3)
	for index, fieldError := range errors {
		if index == 3 {
			break
		}

		field := strings.ToLower(fieldError.Field())
		switch fieldError.Tag() {
		case "required":
			details = append(details, field+" wajib diisi")
		case "min", "gte":
			details = append(details, fmt.Sprintf("%s minimal %s", field, fieldError.Param()))
		case "max", "lte":
			details = append(details, fmt.Sprintf("%s maksimal %s", field, fieldError.Param()))
		case "email":
			details = append(details, field+" harus berupa email yang valid")
		default:
			details = append(details, field+" tidak valid")
		}
	}

	if len(details) == 0 {
		return pkgErr.MsgInvalidRequest
	}

	return "Data permintaan tidak valid: " + strings.Join(details, ", ") + "."
}

func requestID(c echo.Context) string {
	if id := c.Response().Header().Get(config.HeaderRequestID); id != "" {
		return id
	}

	return c.Request().Header.Get(config.HeaderRequestID)
}
