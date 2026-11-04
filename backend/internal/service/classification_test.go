package service

import (
	"testing"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
)

func TestCashSaleIsNotLeftInReview(t *testing.T) {
	sale := entry("sale", "INV-C1", "Budi", 60000, "tunai", "2026-11-04")
	if items := exceptions([]models.Entry{sale}, nil); len(items) != 0 {
		t.Fatalf("exceptions = %d, want 0 for settled cash", len(items))
	}
}

func TestFeeHintForQrisShortfall(t *testing.T) {
	sale := entry("sale", "INV-2", "Rani", 150000, "qris", "2026-11-01")
	receipt := entry("receipt", "INV-2", "QRIS INV-002 RANI", 143500, "qris", "2026-11-01")
	items := exceptions([]models.Entry{sale, receipt}, nil)
	for _, item := range items {
		if item.Entry.Kind != "sale" || item.Entry.ID != sale.ID {
			continue
		}
		if item.FeeHint == nil {
			t.Fatal("want fee hint for shortfall")
		}
		if item.FeeHint.Amount != 6500 || item.FeeHint.Source != "bank" {
			t.Fatalf("hint = %+v", item.FeeHint)
		}
		return
	}
	t.Fatal("sale exception not found")
}

func TestFeeHintMarketplaceSource(t *testing.T) {
	sale := entry("sale", "INV-3", "Toko", 100000, "marketplace", "2026-11-01")
	receipt := entry("receipt", "INV-3", "PENCAIRAN SHOPEE", 90000, "marketplace", "2026-11-01")
	items := exceptions([]models.Entry{sale, receipt}, nil)
	for _, item := range items {
		if item.Entry.ID == sale.ID && item.FeeHint != nil {
			if item.FeeHint.Source != "marketplace" {
				t.Fatalf("source = %q, want marketplace", item.FeeHint.Source)
			}
			return
		}
	}
	t.Fatal("marketplace fee hint not found")
}

func TestSuggestionsSkipCashAndNonOperational(t *testing.T) {
	cash := entry("sale", "INV-1", "Budi", 100000, "tunai", "2026-11-03")
	if matches := suggestions([]models.Entry{cash}, nil); len(matches) != 0 {
		t.Fatalf("cash suggestions = %d, want 0", len(matches))
	}
	sale := entry("sale", "INV-2", "Budi", 2500, "transfer", "2026-11-03")
	interest := entry("receipt", "INV-2", "BUNGA TABUNGAN", 2500, "transfer", "2026-11-03")
	if matches := suggestions([]models.Entry{sale, interest}, nil); len(matches) != 0 {
		t.Fatalf("non-operational suggestions = %d, want 0", len(matches))
	}
}

func TestValidateAllocationRejectsSettledCash(t *testing.T) {
	sale := entry("sale", "INV-1", "Budi", 100000, "tunai", "2026-11-03")
	receipt := entry("receipt", "INV-1", "Budi", 100000, "transfer", "2026-11-03")
	if err := validateAllocation(sale, receipt, nil, 100000, 0, "uji"); err == nil {
		t.Fatal("want error when allocating a settled cash sale")
	}
}

func TestValidateAllocationRejectsNonOperationalReceipt(t *testing.T) {
	sale := entry("sale", "INV-2", "Budi", 2500, "transfer", "2026-11-03")
	receipt := entry("receipt", "", "BUNGA TABUNGAN", 2500, "transfer", "2026-11-03")
	if err := validateAllocation(sale, receipt, nil, 2500, 0, "uji"); err == nil {
		t.Fatal("want error when allocating a non-operational receipt")
	}
}

func TestNonOperationalReceiptIsSkipped(t *testing.T) {
	receipt := entry("receipt", "", "BUNGA TABUNGAN", 2500, "transfer", "2026-11-06")
	if items := exceptions([]models.Entry{receipt}, nil); len(items) != 0 {
		t.Fatalf("exceptions = %d, want 0 for interest", len(items))
	}
}

func TestAutoMatchSkipsNonOperationalReceipt(t *testing.T) {
	sale := entry("sale", "INV-9", "Budi", 6500, "transfer", "2026-11-03")
	receipt := entry("receipt", "INV-9", "TRANSFER ANTAR REKENING BUDI", 6500, "transfer", "2026-11-03")
	pairs := autoMatchPairs([]models.Entry{sale, receipt}, 0.95)
	if len(pairs) != 0 {
		t.Fatalf("pairs = %d, want 0 for non-operational receipt", len(pairs))
	}
}

func TestNonOperationalDoesNotHideCustomers(t *testing.T) {
	for _, name := range []string{"Bunga Pertiwi", "Toko Pajak", "TRF BUNGA ANGGRAINI"} {
		receipt := entry("receipt", "INV-1", name, 100000, "transfer", "2026-11-03")
		if isNonOperationalReceipt(receipt) {
			t.Fatalf("%q wrongly classified as non-operational", name)
		}
	}
}

func TestCashChannelVariants(t *testing.T) {
	for _, channel := range []string{"tunai", "Tunai", "Cash", "Tunai (Cash)", "Bayar Tunai"} {
		sale := entry("sale", "INV-1", "Budi", 10000, channel, "2026-11-03")
		if !isCashSale(sale) {
			t.Fatalf("channel %q should be treated as cash", channel)
		}
	}
	if isCashSale(entry("sale", "INV-1", "Budi", 10000, "cashless", "2026-11-03")) {
		t.Fatal("cashless should not be treated as cash")
	}
}
