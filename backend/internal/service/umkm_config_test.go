package service

import (
	"testing"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
)

func TestConfigExposesLimits(t *testing.T) {
	svc := NewService(config.Config{}, nil, nil)
	got := svc.Config()
	want := models.Limits{MaxImportRows: MaxImportRows, MaxCSVBytes: MaxCSVBytes, MaxEntries: MaxEntries}
	if got.Limits != want {
		t.Fatalf("config limits = %+v, want %+v", got.Limits, want)
	}
}
