package service

import (
	"testing"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
)

func TestParseRupiahLocales(t *testing.T) {
	cases := []struct {
		raw  string
		want int64
	}{
		{"Rp1.250.000", 1250000},
		{"1,250.00", 1250},
		{"1.234,56", 1235},
		{"1.250", 1250},
		{"(500.000)", -500000},
	}
	for _, tc := range cases {
		got, err := parseRupiah(tc.raw)
		if err != nil || got != tc.want {
			t.Fatalf("parseRupiah(%q) = %d, %v; want %d", tc.raw, got, err, tc.want)
		}
	}
	if _, err := parseRupiah("(1.000"); err == nil {
		t.Fatal("want error for unbalanced parentheses")
	}
}

func TestExtractReferenceFromDescription(t *testing.T) {
	if got := extractReference("TRF DARI BUDI INV-102 LUNAS"); got != "INV-102" {
		t.Fatalf("reference = %q", got)
	}
}

func TestAutoMatchPairsRejectsDifferentName(t *testing.T) {
	sale := entry("sale", "INV-9", "Budi", 100000, "transfer", "2026-11-03")
	receipt := entry("receipt", "INV-9", "Siti", 100000, "transfer", "2026-11-03")
	pairs := autoMatchPairs([]models.Entry{sale, receipt}, 0.95)
	if len(pairs) != 0 {
		t.Fatalf("pairs = %d, want 0 when names differ", len(pairs))
	}
}
