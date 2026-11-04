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

func TestClearWorkspaceDataKeepsAccountAndAudit(t *testing.T) {
	t.Setenv("MATCH_AUTO_APPLY", "false")
	db := testkit.Prepare(t)
	cfg := config.Config{
		App:    config.AppConfig{Name: "clear-data-test"},
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second},
	}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}

	password := "clear-password-2026"
	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "Pemilik Clear", Email: "clear-" + randomHex() + "@example.test", Password: password, Workspace: "Toko Clear",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	saleCSV := "id,tanggal,referensi,nama,nominal,kanal,catatan\nC-1,2026-11-01,INV-C,Ayu,100000,transfer,"
	receiptCSV := "id,tanggal,referensi,nama,nominal,kanal,catatan\nCR-1,2026-11-02,INV-C,Ayu,100000,transfer,"
	requireStatus(t, client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: saleCSV}, true), http.StatusCreated)
	requireStatus(t, client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "receipt", CSV: receiptCSV}, true), http.StatusCreated)

	var matches []models.Match
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/suggestions", nil, false), &matches)
	if len(matches) != 1 {
		t.Fatalf("suggestions = %+v, want one pair", matches)
	}
	allocation := models.AllocationRequest{
		SaleID: matches[0].Sale.ID, ReceiptID: matches[0].Receipt.ID,
		Amount: 100000, Fee: 0, Reason: "pengujian hapus", Key: randomHex(),
	}
	requireStatus(t, client.request(http.MethodPost, "/api/v1/umkm/allocations", allocation, true), http.StatusCreated)

	badConfirm := client.request(http.MethodDelete, "/api/v1/umkm/data", models.ClearWorkspaceDataRequest{Password: password, Confirm: "JANGAN"}, true)
	requireStatus(t, badConfirm, http.StatusBadRequest)
	badPassword := client.request(http.MethodDelete, "/api/v1/umkm/data", models.ClearWorkspaceDataRequest{Password: "salah-total-2026", Confirm: "HAPUS"}, true)
	requireStatus(t, badPassword, http.StatusBadRequest)
	cleared := client.request(http.MethodDelete, "/api/v1/umkm/data", models.ClearWorkspaceDataRequest{Password: password, Confirm: "HAPUS"}, true)
	requireStatus(t, cleared, http.StatusOK)

	var state models.StateResponse
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/state", nil, false), &state)
	if len(state.State.Entries) != 0 || len(state.State.Imports) != 0 || len(state.State.Allocations) != 0 {
		t.Fatalf("data masih tersisa: %+v", state.State)
	}
	if len(state.Users) != 1 || state.Events[0].Action != "Kosongkan data workspace" {
		t.Fatalf("akun/audit tidak dipertahankan: users=%d events=%+v", len(state.Users), state.Events)
	}

	// The checksum and import batch must also be gone, otherwise the same CSV
	// would be rejected as a duplicate after the user starts over.
	reimport := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: saleCSV}, true)
	requireStatus(t, reimport, http.StatusCreated)
	var result models.CommitImportResponse
	decode(t, reimport, &result)
	if result.Imported != 1 || result.Duplicate {
		t.Fatalf("reimport after clear = %+v, want one new row", result)
	}
}
