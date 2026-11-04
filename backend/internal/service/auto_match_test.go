package service

import (
	"testing"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
)

func entry(kind, ref, name string, amount int64, channel, date string) models.Entry {
	return models.Entry{ID: newID(), Kind: kind, Reference: ref, Name: name, Amount: amount, Channel: channel, Date: date}
}

func TestScorePairPerfectMatch(t *testing.T) {
	sale := entry("sale", "INV-102", "Rani", 100000, "qris", "2026-11-03")
	receipt := entry("receipt", "inv102", "RANI", 100000, "QRIS", "2026-11-03")
	score, reasons := scorePair(sale, receipt)
	if score < 0.999 {
		t.Fatalf("score = %.3f, want 1.0 (reasons %v)", score, reasons)
	}
}

func TestScorePairPartialReferenceDrops(t *testing.T) {
	sale := entry("sale", "TRF/INV-102", "Rani", 100000, "qris", "2026-11-03")
	receipt := entry("receipt", "INV102", "Rani", 100000, "qris", "2026-11-04")
	score, _ := scorePair(sale, receipt)
	if score >= 0.95 {
		t.Fatalf("score = %.3f, want below auto threshold", score)
	}
	if score < 0.75 {
		t.Fatalf("score = %.3f, want a review candidate", score)
	}
}

func TestAutoMatchPairsSkipsAmbiguousReceipts(t *testing.T) {
	sale := entry("sale", "INV-1", "Budi", 50000, "transfer", "2026-11-03")
	r1 := entry("receipt", "INV-1", "Budi", 50000, "transfer", "2026-11-03")
	r2 := entry("receipt", "INV-1", "Budi", 50000, "transfer", "2026-11-03")
	pairs := autoMatchPairs([]models.Entry{sale, r1, r2}, 0.95)
	if len(pairs) != 0 {
		t.Fatalf("pairs = %d, want 0 for ambiguous receipt", len(pairs))
	}
}

func TestAutoMatchPairsSkipsAmountMismatch(t *testing.T) {
	sale := entry("sale", "INV-9", "Sari", 75000, "qris", "2026-11-03")
	receipt := entry("receipt", "INV-9", "Sari", 70000, "qris", "2026-11-03")
	pairs := autoMatchPairs([]models.Entry{sale, receipt}, 0.95)
	if len(pairs) != 0 {
		t.Fatalf("pairs = %d, want 0 for amount mismatch", len(pairs))
	}
}

func TestAutoMatchPairsAcceptsCleanPair(t *testing.T) {
	sale := entry("sale", "INV-7", "Dewi", 120000, "transfer", "2026-11-03")
	receipt := entry("receipt", "INV-7", "Dewi", 120000, "transfer", "2026-11-03")
	pairs := autoMatchPairs([]models.Entry{sale, receipt}, 0.95)
	if len(pairs) != 1 {
		t.Fatalf("pairs = %d, want 1", len(pairs))
	}
	if pairs[0].sale.ID != sale.ID || pairs[0].receipt.ID != receipt.ID {
		t.Fatalf("unexpected pair")
	}
}

func TestNormalizeReference(t *testing.T) {
	if got := normalizeReference(" TRF/INV-102 "); got != "trfinv102" {
		t.Fatalf("normalizeReference = %q", got)
	}
}
