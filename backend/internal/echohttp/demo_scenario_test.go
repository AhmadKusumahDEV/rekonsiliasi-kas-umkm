package echohttp_test

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/testkit"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile("../testdata/imports/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(body)
}

func TestDemoScenarioBankVersusSales(t *testing.T) {
	t.Setenv("MATCH_AUTO_APPLY", "true")
	t.Setenv("MATCH_AUTO_THRESHOLD", "0.95")
	db := testkit.Prepare(t)
	cfg := config.Config{
		App:    config.AppConfig{Name: "demo-scenario"},
		Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second},
	}
	svc := service.NewService(cfg, db, nil)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}

	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "Demo Scenario", Email: "demo-" + randomHex() + "@example.test", Password: "demo-scenario-2026", Workspace: "Demo Kas",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	requireStatus(t, client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{
		Kind: "sale", CSV: readFixture(t, "demo_sales.csv"),
	}, true), http.StatusCreated)

	flex := client.request(http.MethodPost, "/api/v1/umkm/imports/flexible", models.FlexibleImportRequest{
		Kind: "receipt", CSV: readFixture(t, "demo_bank.csv"),
	}, true)
	requireStatus(t, flex, http.StatusOK)
	var normalized models.FlexibleImportResponse
	decode(t, flex, &normalized)
	if len(normalized.Warnings) == 0 {
		t.Fatalf("expected warning for outgoing bank row")
	}

	commit := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{
		Kind: "receipt", CSV: normalized.CSV,
	}, true)
	requireStatus(t, commit, http.StatusCreated)
	var result models.CommitImportResponse
	decode(t, commit, &result)

	var state models.StateResponse
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/state", nil, false), &state)
	var exceptions []models.Exception
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/exceptions", nil, false), &exceptions)
	cashSettled, nonOperational := false, false
	for _, item := range state.State.Entries {
		if item.SourceID == "S09" && item.Settled {
			cashSettled = true
		}
		if item.Category == "non_operational" {
			nonOperational = true
		}
	}
	if !cashSettled {
		t.Fatal("cash sale S09 should be marked settled")
	}
	if !nonOperational {
		t.Fatal("bank interest should be categorised non_operational")
	}

	t.Logf("rows imported=%d skipped=%d auto_matched=%d", result.Imported, result.Skipped, result.AutoMatched)
	t.Logf("entries=%d allocations=%d exceptions=%d warnings=%v",
		len(state.State.Entries), len(state.State.Allocations), len(exceptions), normalized.Warnings)
	for _, item := range exceptions {
		if item.Entry.Category == "cash" || item.Entry.Category == "non_operational" {
			t.Fatalf("categorised entry leaked into review: %+v", item.Entry)
		}
		t.Logf("exception status=%s ref=%q name=%q amount=%d remaining=%d reason=%q",
			item.Status, item.Entry.Reference, item.Entry.Name, item.Entry.Amount, item.Remaining, item.Reason)
	}
	for _, item := range exceptions {
		if item.Entry.SourceID == "S02" {
			if item.FeeHint == nil || item.FeeHint.Amount != 6500 || item.FeeHint.Source != "bank" {
				t.Fatalf("INV-002 fee hint = %+v, want 6500 from bank", item.FeeHint)
			}
		}
	}

	if result.AutoMatched < 3 {
		t.Fatalf("auto_matched = %d, want at least 3 clean pairs", result.AutoMatched)
	}
	if len(state.State.Allocations) < 3 {
		t.Fatalf("allocations = %d, want at least 3", len(state.State.Allocations))
	}
	if len(exceptions) == 0 {
		t.Fatal("expected deliberate discrepancies to reach the review queue")
	}
}
