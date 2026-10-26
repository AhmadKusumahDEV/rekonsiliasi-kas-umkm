package echohttp_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/labstack/echo/v4"
)

func testServer(t *testing.T) *echo.Echo {
	t.Helper()

	cfg := config.Config{
		App: config.AppConfig{Name: "test", Environment: "test"},
		Server: config.ServerConfig{
			Host:            "127.0.0.1",
			Port:            8080,
			ShutdownTimeout: time.Second,
			AllowedOrigins:  []string{"http://localhost:5173"},
		},
		Database: config.DatabaseConfig{Enabled: false},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewService(cfg, nil, nil)
	return echohttp.NewEcho(cfg, svc, logger)
}

func TestLiveness(t *testing.T) {
	server := testServer(t)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status body = %v, want ok", body["status"])
	}
}

func TestReadinessWithoutDatabase(t *testing.T) {
	server := testServer(t)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body struct {
		Status string `json:"status"`
		Checks map[string]struct {
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Checks["postgres"].Status != "disabled" {
		t.Fatalf("postgres status = %q, want disabled", body.Checks["postgres"].Status)
	}
}

func TestNotFoundUsesJSONErrorEnvelope(t *testing.T) {
	server := testServer(t)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tidak-ada", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != http.StatusNotFound || body.Message == "" {
		t.Fatalf("unexpected body: %+v", body)
	}
}
