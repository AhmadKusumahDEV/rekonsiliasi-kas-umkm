package echohttp_test

import (
	"fmt"
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
	requireStatus(t, client.request(http.MethodGet, "/api/v1/umkm/exceptions", nil, false), http.StatusUnauthorized)
	requireStatus(t, client.request(http.MethodGet, "/api/v1/umkm/audit", nil, false), http.StatusUnauthorized)
	registered := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{Name: "Pemilik Audit", Email: "audit-" + randomHex() + "@example.test", Password: "audit-password-2026", Workspace: "Toko Audit"}, false)
	requireStatus(t, registered, http.StatusCreated)
	client.captureSession(registered)
	header := "id,tanggal,referensi,nama,nominal,kanal,catatan\n"
	row := "P101,2026-10-28,INV-1,Ayu,150000,transfer,"
	csv := header + row
	requireStatus(t, client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: csv}, true), http.StatusCreated)
	var before models.StateResponse
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/state", nil, false), &before)
	if len(before.Events) == 0 || before.Events[0].EntityType == nil || *before.Events[0].EntityType != "import" || before.Events[0].EntityID == nil || len(*before.Events[0].EntityID) != 32 {
		t.Fatalf("import audit is not linked to its batch: %+v", before.Events)
	}
	var audit []models.AuditEvent
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/audit?limit=1", nil, false), &audit)
	if len(audit) != 1 || audit[0].ID != before.Events[0].ID {
		t.Fatalf("audit page = %+v, want newest event %d", audit, before.Events[0].ID)
	}
	var older []models.AuditEvent
	decode(t, client.request(http.MethodGet, fmt.Sprintf("/api/v1/umkm/audit?before_id=%d&limit=1", audit[0].ID), nil, false), &older)
	if len(older) != 0 {
		t.Fatalf("audit cursor returned newer/duplicate rows: %+v", older)
	}
	requireStatus(t, client.request(http.MethodGet, "/api/v1/umkm/audit?limit=101", nil, false), http.StatusBadRequest)
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

	var exceptions []models.Exception
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/exceptions", nil, false), &exceptions)
	if len(exceptions) != 1 || exceptions[0].Status != "unmatched" || exceptions[0].Remaining != 150000 || exceptions[0].CandidateCount != 0 || exceptions[0].SuggestionAvailable {
		t.Fatalf("unexpected exceptions: %+v", exceptions)
	}
}
