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

func TestAutoMatchAppliesCleanPair(t *testing.T) {
	t.Setenv("MATCH_AUTO_APPLY", "true")
	t.Setenv("MATCH_AUTO_THRESHOLD", "0.95")
	db := testkit.Prepare(t)
	cfg := config.Config{
		App:    config.AppConfig{Name: "auto-match-test"},
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second},
	}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}

	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "Pemilik Auto", Email: "auto-" + randomHex() + "@example.test", Password: "auto-match-password-2026", Workspace: "Toko Auto",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	saleCSV := "id,tanggal,referensi,nama,nominal,kanal,catatan\nS-1,2026-11-03,INV-77,Budi,150000,transfer,"
	receiptCSV := "id,tanggal,referensi,nama,nominal,kanal,catatan\nR-1,2026-11-03,INV-77,Budi,150000,transfer,"
	requireStatus(t, client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: saleCSV}, true), http.StatusCreated)
	imported := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "receipt", CSV: receiptCSV}, true)
	requireStatus(t, imported, http.StatusCreated)
	var commit models.CommitImportResponse
	decode(t, imported, &commit)
	if commit.AutoMatched != 1 {
		t.Fatalf("auto_matched = %d, want 1", commit.AutoMatched)
	}

	var state models.StateResponse
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/state", nil, false), &state)
	if len(state.State.Allocations) != 1 {
		t.Fatalf("allocations = %d, want 1", len(state.State.Allocations))
	}
	var suggestions []models.Match
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/suggestions", nil, false), &suggestions)
	if len(suggestions) != 0 {
		t.Fatalf("suggestions = %d, want 0 after auto match", len(suggestions))
	}
}
