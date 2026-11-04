package echohttp_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/ai"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/echohttp"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/testkit"
)

type fakeReviewer struct {
	calls int
}

func (f *fakeReviewer) Review(_ context.Context, candidates []ai.Candidate) ([]ai.Verdict, error) {
	f.calls++
	verdicts := make([]ai.Verdict, 0, len(candidates))
	for _, candidate := range candidates {
		verdicts = append(verdicts, ai.Verdict{SaleID: candidate.SaleID, ReceiptID: candidate.ReceiptID, Match: true, Reason: "nominal dan referensi sesuai"})
	}
	return verdicts, nil
}

func TestAIReviewerPromotesAmbiguousPair(t *testing.T) {
	t.Setenv("MATCH_AUTO_APPLY", "true")
	t.Setenv("MATCH_AUTO_THRESHOLD", "0.95")
	t.Setenv("AI_MIN_SCORE", "0.90")
	db := testkit.Prepare(t)
	cfg := config.Config{App: config.AppConfig{Name: "ai-match"}, Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ShutdownTimeout: time.Second}}
	svc := service.NewService(cfg, db, nil)
	reviewer := &fakeReviewer{}
	svc.SetAIReviewer(reviewer)
	server := &echohttpServer{handler: echohttp.NewEcho(cfg, svc, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	client := &testClient{t: t, server: server}

	register := client.request(http.MethodPost, "/api/v1/auth/register", models.RegisterRequest{
		Name: "AI Match", Email: "ai-" + randomHex() + "@example.test", Password: "ai-match-password-2026", Workspace: "Toko AI",
	}, false)
	requireStatus(t, register, http.StatusCreated)
	client.captureSession(register)

	saleCSV := "id,tanggal,referensi,nama,nominal,kanal,catatan\nS-AI,2026-11-03,INV-9,Budi,100000,transfer,"
	receiptCSV := "id,tanggal,referensi,nama,nominal,kanal,catatan\nR-AI,2026-11-03,INV-9,Siti,100000,transfer,"
	requireStatus(t, client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "sale", CSV: saleCSV}, true), http.StatusCreated)
	commit := client.request(http.MethodPost, "/api/v1/umkm/imports", models.ImportRequest{Kind: "receipt", CSV: receiptCSV}, true)
	requireStatus(t, commit, http.StatusCreated)
	var result models.CommitImportResponse
	decode(t, commit, &result)
	if result.AIMatched != 1 || result.AutoMatched != 0 {
		t.Fatalf("auto=%d ai=%d, want 0 rule and 1 AI", result.AutoMatched, result.AIMatched)
	}
	if reviewer.calls == 0 {
		t.Fatal("reviewer was not called")
	}

	var state models.StateResponse
	decode(t, client.request(http.MethodGet, "/api/v1/umkm/state", nil, false), &state)
	if len(state.State.Allocations) != 1 {
		t.Fatalf("allocations = %d, want 1", len(state.State.Allocations))
	}
	if !strings.Contains(state.State.Allocations[0].Reason, "Disarankan AI") {
		t.Fatalf("reason = %q", state.State.Allocations[0].Reason)
	}
}
