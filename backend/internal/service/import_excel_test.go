package service

import (
	"context"
	"strings"
	"testing"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
)

func TestUploadRejectsLegacyAndPDF(t *testing.T) {
	svc := &Service{}
	user := models.User{}
	if _, err := svc.ImportUploadedFile(context.Background(), user, "sale", "", "lama.xls", []byte{0xD0, 0xCF, 0x11, 0xE0}); err == nil || !strings.Contains(err.Error(), ".xls") {
		t.Fatalf("legacy err = %v", err)
	}
	if _, err := svc.ImportUploadedFile(context.Background(), user, "sale", "", "mutasi.pdf", []byte("%PDF-1.7")); err == nil || !strings.Contains(err.Error(), "PDF") {
		t.Fatalf("pdf err = %v", err)
	}
}

func TestUploadRejectsEmptyAndOversize(t *testing.T) {
	svc := &Service{}
	user := models.User{}
	if _, err := svc.ImportUploadedFile(context.Background(), user, "sale", "", "kosong.csv", nil); err == nil {
		t.Fatal("want error for empty file")
	}
	big := make([]byte, maxUploadBytes+1)
	if _, err := svc.ImportUploadedFile(context.Background(), user, "sale", "", "besar.csv", big); err == nil || !strings.Contains(err.Error(), "10 MB") {
		t.Fatalf("oversize err = %v", err)
	}
}

func TestUploadXLSXDisabledByFlag(t *testing.T) {
	t.Setenv("IMPORT_XLSX_ENABLED", "false")
	svc := &Service{}
	if _, err := svc.ImportUploadedFile(context.Background(), models.User{}, "sale", "", "x.xlsx", []byte("PK\x03\x04x")); err == nil || !strings.Contains(err.Error(), "belum diaktifkan") {
		t.Fatalf("disabled err = %v", err)
	}
}

func TestDetectHeaderRow(t *testing.T) {
	rows := [][]string{{"Rekap"}, {"Tanggal", "Nama", "Nominal"}, {"2026-11-03", "Budi", "100000"}}
	if got := detectHeaderRow(rows); got != 2 {
		t.Fatalf("header row = %d, want 2", got)
	}
}
