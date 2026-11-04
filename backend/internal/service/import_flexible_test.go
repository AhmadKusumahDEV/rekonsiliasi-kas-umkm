package service

import (
	"strings"
	"testing"
)

func TestNormalizeFlexibleSemicolonAndRupiah(t *testing.T) {
	raw := "Tgl;Keterangan;Jumlah;Metode\n" +
		"03/11/2026;Penjualan kue;Rp1.250.000;Transfer\n"
	canonical, warnings, err := normalizeFlexibleCSV("sale", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if !strings.Contains(canonical, "1250000") {
		t.Fatalf("canonical missing amount: %s", canonical)
	}
	if !strings.Contains(canonical, "2026-11-03") {
		t.Fatalf("canonical missing normalized date: %s", canonical)
	}
	if !strings.Contains(canonical, "transfer") {
		t.Fatalf("canonical missing channel: %s", canonical)
	}
}

func TestFlexibleFindsHeaderAfterMetadata(t *testing.T) {
	raw := "REKENING KORAN BCA\n" +
		"Periode: 01/11/2026 - 30/11/2026\n" +
		"\n" +
		"Tanggal,Keterangan,Debit,Kredit,Saldo\n" +
		"2026-11-03,TRF INV-1,0,100000,1000000\n"
	canonical, warnings, err := normalizeFlexibleCSV("receipt", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if !strings.Contains(canonical, "100000") || !strings.Contains(canonical, "2026-11-03") {
		t.Fatalf("canonical missing row: %s", canonical)
	}
}

func TestNormalizeFlexibleKeepsExplicitID(t *testing.T) {
	raw := "id,tanggal,keterangan,nominal,metode\n" +
		"SRC-1,2026-11-03,Budi,150000,Transfer\n"
	canonical, _, err := normalizeFlexibleCSV("sale", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if !strings.Contains(canonical, "SRC-1") {
		t.Fatalf("explicit id not preserved: %s", canonical)
	}
}

func TestFlexibleImportAllRowsSkippedReturnsWarnings(t *testing.T) {
	raw := "tanggal,keterangan,debit,kredit\n" +
		"2026-11-03,BIAYA ADMIN,6500,0\n"
	svc := &Service{}
	preview, canonical, warnings, err := svc.FlexibleImport("receipt", raw)
	if err != nil {
		t.Fatalf("flexible import: %v", err)
	}
	if preview.Count != 0 || len(preview.Rows) != 0 {
		t.Fatalf("preview = %+v, want empty", preview)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want one", warnings)
	}
	if strings.Count(strings.TrimSpace(canonical), "\n") != 0 {
		t.Fatalf("canonical should contain only the header: %q", canonical)
	}
}

func TestNormalizeFlexibleReceiptSkipsOutgoing(t *testing.T) {
	raw := "tanggal,keterangan,debit,kredit\n" +
		"2026-11-03,BIAYA ADMIN,6500,0\n" +
		"2026-11-03,TRF BUDI,0,150000\n"
	canonical, warnings, err := normalizeFlexibleCSV("receipt", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 skipped row", warnings)
	}
	if strings.Contains(canonical, "6500") {
		t.Fatalf("outgoing row leaked into receipts: %s", canonical)
	}
	if !strings.Contains(canonical, "150000") {
		t.Fatalf("credit row missing: %s", canonical)
	}
}

func TestNormalizeFlexibleMissingColumn(t *testing.T) {
	raw := "tanggal,keterangan\n2026-11-03,tes\n"
	if _, _, err := normalizeFlexibleCSV("sale", raw); err == nil {
		t.Fatal("want error for missing nominal")
	}
}
