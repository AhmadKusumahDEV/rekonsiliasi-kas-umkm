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

func TestDuplicateImportDoesNotMutateAndPreviewUsesWorkspace(t *testing.T) {
	db := testkit.Prepare(t)
	cfg := config.Config{App: config.AppConfig{Name: "audit-test"}, Server: config.ServerConfig{ShutdownTimeout: time.Second}}
	svc := service.NewService(cfg, db, nil)
	client := &testClient{t: t, server: &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}}
	registered := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{Name: "Pemilik Audit", Email: "audit-" + randomHex() + "@example.test", Password: "audit-password-2026", Workspace: "Toko Audit"}, false)
	requireStatus(t, registered, http.StatusCreated)
	client.captureSession(registered)
	header := "id,tanggal,referensi,nama,nominal,kanal,catatan\n"
	row := "P101,2026-10-28,INV-1,Ayu,150000,transfer,"
	csv := header + row
	requireStatus(t, client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: csv}, true), http.StatusCreated)
	var before models.StateResponse
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/state", nil, false), &before)
	for _, payload := range []string{csv, csv + "\n"} {
		response := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: payload}, true)
		requireStatus(t, response, http.StatusCreated)
		var result models.CommitImportResponse
		decode(t, response, &result)
		if !result.Duplicate || result.Imported != 0 || result.Skipped != 1 {
			t.Fatalf("unexpected no-op result: %+v", result)
		}
		var after models.StateResponse
		decode(t, client.request(http.MethodGet, "/api/v1/umkm/state", nil, false), &after)
		if after.Workspace.Version != before.Workspace.Version || len(after.Events) != len(before.Events) || len(after.State.Imports) != 1 {
			t.Fatalf("no-op import changed version, audit or batches: before=%d after=%d", before.Workspace.Version, after.Workspace.Version)
		}
	}
	preview := client.request(http.MethodPost, "/api/v1/umkm/imports/preview", models.ImportRequest{Kind: "sale", CSV: csv + "\nS2,2026-10-28,INV-2,Ayu,50000,transfer,"}, true)
	requireStatus(t, preview, http.StatusOK)
	var result models.PreviewResponse
	decode(t, preview, &result)
	if result.Count != 1 || result.Skipped != 1 || result.Total != 50000 || len(result.Rows) != 1 || result.Rows[0].SourceID != "S2" {
		t.Fatalf("preview counted an existing entry: %+v", result)
	}
	conflict := client.request(http.MethodPost, "/api/v1/umkm/imports/preview", models.ImportRequest{Kind: "sale", CSV: header + "P101,2026-10-28,INV-1,Ayu,999,transfer,"}, true)
	requireStatus(t, conflict, http.StatusConflict)
}
