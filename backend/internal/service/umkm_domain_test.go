package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
)

func TestParseCSV(t *testing.T) {
	raw := "\ufeffid,tanggal,referensi,nama,nominal,kanal,catatan\nS1,2026-10-01,INV-1,Ayu,150000,Transfer,tes\nS2,2026-10-02,INV-2,Bima,75000,qris,"
	entries, total, sum, err := parseCSV("sale", raw)
	if err != nil {
		t.Fatalf("parseCSV error = %v", err)
	}
	if len(entries) != 2 || total != 225000 || sum == "" {
		t.Fatalf("unexpected result: len=%d total=%d checksum=%q", len(entries), total, sum)
	}
	if entries[0].Channel != "transfer" || entries[0].Kind != "sale" || len(entries[0].ID) != 32 {
		t.Fatalf("unexpected entry: %+v", entries[0])
	}
}

func TestParseCSVRejectsInvalidRows(t *testing.T) {
	tests := []string{
		"id,tanggal,referensi,nama,nominal,kanal,catatan\nS1,2026-10-01,REF,Ayu,0,transfer,",
		"id,tanggal,referensi,nama,nominal,kanal,catatan\nS1,2026-10-01,REF,Ayu,1000,transfer,\nS1,2026-10-02,REF,Bima,2000,qris,",
		"id,tanggal,referensi,nama,nominal,kanal,catatan\nS1,2026-10-01,REF,Ayu,abc,transfer,",
	}
	for _, raw := range tests {
		if _, _, _, err := parseCSV("sale", raw); err == nil {
			t.Fatalf("expected invalid CSV error for %q", raw)
		}
	}
}

func TestSuggestionsRequireUniqueCandidateInBothDirections(t *testing.T) {
	sale := models.Entry{ID: "sale-1", Kind: "sale", Date: "2026-10-01", Reference: "INV-1", Channel: "transfer", Amount: 100000, Name: "A"}
	receipt := models.Entry{ID: "receipt-1", Kind: "receipt", Date: "2026-10-03", Reference: "inv-1", Channel: "transfer", Amount: 100000, Name: "A"}

	matches := suggestions([]models.Entry{sale, receipt}, nil)
	if len(matches) != 1 || matches[0].Sale.ID != sale.ID || matches[0].Receipt.ID != receipt.ID {
		t.Fatalf("unexpected matches: %+v", matches)
	}

	late := receipt
	late.ID = "receipt-2"
	late.Date = "2026-10-05"
	if matches := suggestions([]models.Entry{sale, late}, nil); len(matches) != 0 {
		t.Fatalf("receipt outside 0-3 day window matched: %+v", matches)
	}

	second := receipt
	second.ID = "receipt-3"
	second.Date = "2026-10-04"
	if matches := suggestions([]models.Entry{sale, receipt, second}, nil); len(matches) != 0 {
		t.Fatalf("ambiguous pair matched: %+v", matches)
	}
}

func TestAllocationRemainingAndValidation(t *testing.T) {
	sale := models.Entry{ID: "a", Kind: "sale", Amount: 200000}
	receipt := models.Entry{ID: "b", Kind: "receipt", Amount: 190000}
	active := []models.Allocation{}

	if err := validateAllocation(sale, receipt, active, 180000, 10000, "biaya admin"); err != nil {
		t.Fatalf("valid allocation rejected: %v", err)
	}
	if err := validateAllocation(sale, receipt, active, 191000, 10000, "biaya admin"); !errors.Is(err, pkgErr.ErrConflict) {
		t.Fatalf("over-allocation error = %v, want conflict", err)
	}
	if err := validateAllocation(sale, receipt, active, 1000, 1000, "x"); !errors.Is(err, pkgErr.ErrInvalidRequest) {
		t.Fatalf("short fee reason error = %v, want invalid request", err)
	}

	allocated := []models.Allocation{{SaleID: sale.ID, ReceiptID: receipt.ID, Amount: 100000, Fee: 5000}}
	if got := remaining(sale, allocated); got != 95000 {
		t.Fatalf("sale remaining = %d, want 95000", got)
	}
	if got := remaining(receipt, allocated); got != 90000 {
		t.Fatalf("receipt remaining = %d, want 90000", got)
	}
}

func TestValidHexIDRequiresLowercase(t *testing.T) {
	if !validHexID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Fatal("valid lowercase ID rejected")
	}
	if validHexID("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA") || validHexID("short") {
		t.Fatal("invalid ID accepted")
	}
}

func TestExportNeutralizesSpreadsheetFormulas(t *testing.T) {
	entries := []models.Entry{{
		ID:        "1",
		SourceID:  "S1",
		Kind:      "sale",
		Date:      "2026-10-01",
		Reference: "=HYPERLINK(\"https://example.test\")",
		Name:      "Toko",
		Amount:    1000,
		Channel:   "tunai",
		Note:      "+catatan",
	}}
	data, err := buildExportCSV(entries, nil)
	if err != nil {
		t.Fatalf("buildExportCSV error = %v", err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "\ufeff") {
		t.Fatal("export has no UTF-8 BOM")
	}
	if !strings.Contains(text, "'=HYPERLINK") || !strings.Contains(text, "'+catatan") {
		t.Fatalf("formula was not neutralized: %s", text)
	}
}
