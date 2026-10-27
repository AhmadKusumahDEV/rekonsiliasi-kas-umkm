package echohttp_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
)

func TestPublicConfigAndValidationWithoutDatabase(t *testing.T) {
	cfg := config.Config{
		App:    config.AppConfig{Name: "validation-test"},
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second},
	}
	svc := service.NewService(cfg, nil, nil)
	e := echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	configRequest := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	configRecorder := httptest.NewRecorder()
	e.ServeHTTP(configRecorder, configRequest)
	if configRecorder.Code != http.StatusOK || !strings.Contains(configRecorder.Body.String(), "Easy Financial Audit") {
		t.Fatalf("config = %d %s", configRecorder.Code, configRecorder.Body.String())
	}

	tests := []struct {
		name string
		path string
		body string
	}{
		{"register empty", "/api/v1/auth/register", `{}`},
		{"register bad email", "/api/v1/auth/register", `{"name":"Ayu","email":"bad","password":"123456789012","workspace":"Toko"}`},
		{"register short password", "/api/v1/auth/register", `{"name":"Ayu","email":"ayu@example.test","password":"short","workspace":"Toko"}`},
		{"login empty", "/api/v1/auth/login", `{}`},
		{"login bad json", "/api/v1/auth/login", `{`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			e.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}

	protected := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	protectedRecorder := httptest.NewRecorder()
	e.ServeHTTP(protectedRecorder, protected)
	if protectedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("protected without cookie = %d, want 401", protectedRecorder.Code)
	}
}
