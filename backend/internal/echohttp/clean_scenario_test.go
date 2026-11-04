package echohttp_test

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/testkit"
)

func TestDemoScenarioClean(t *testing.T) {
	t.Setenv("MATCH_AUTO_APPLY", "true")
	t.Setenv("MATCH_AUTO_THRESHOLD", "0.95")
	db := testkit.Prepare(t)
	cfg := config.Config{App: config.AppConfig{Name: "demo-clean"}, Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second}}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}

	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "Demo Clean", Email: "clean-" + randomHex() + "@example.test", Password: "demo-clean-2026", Workspace: "Demo Bersih",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	requireStatus(t, client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{
		Kind: "sale", CSV: readFixture(t, "demo_sales_clean.csv"),
	}, true), http.StatusCreated)

	flex := client.request(http.MethodPost, "/api/v1/umkm/imports/flexible", models.FlexibleImportRequest{
		Kind: "receipt", CSV: readFixture(t, "demo_bank_clean.csv"),
	}, true)
	requireStatus(t, flex, http.StatusOK)
	var normalized models.FlexibleImportResponse
	decode(t, flex, &normalized)
	if len(normalized.Warnings) != 0 {
		t.Fatalf("warnings = %v, want none for clean credits", normalized.Warnings)
	}

	commit := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "receipt", CSV: normalized.CSV}, true)
	requireStatus(t, commit, http.StatusCreated)
	var result models.CommitImportResponse
	decode(t, commit, &result)
	if result.AutoMatched != 11 {
		t.Fatalf("auto_matched = %d, want 11", result.AutoMatched)
	}

	var state models.StateResponse
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/state", nil, false), &state)
	var exceptions []models.Exception
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/exceptions", nil, false), &exceptions)
	if len(exceptions) != 3 {
		t.Fatalf("exceptions = %d, want 3", len(exceptions))
	}
	for _, item := range exceptions {
		if item.Entry.Category == "non_operational" || item.Entry.Category == "cash" {
			t.Fatalf("unexpected category in review: %+v", item.Entry)
		}
	}
}
