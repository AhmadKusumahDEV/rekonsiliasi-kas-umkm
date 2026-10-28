package config_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
)

func TestLoadFromExampleAndEnvironment(t *testing.T) {
	example := filepath.Join("..", "..", "configs", "local.yaml.example")
	t.Setenv(config.EnvAppConfigFile, example)
	t.Setenv(config.EnvServerPort, "9090")
	t.Setenv(config.EnvDatabaseEnabled, "true")
	t.Setenv(config.EnvDatabaseURL, "postgres://user:pass@localhost:5432/rekonsiliasi?sslmode=disable")
	t.Setenv(config.EnvCORSAllowedOrigins, "http://localhost:3000,https://app.example.com")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("server.port = %d, want 9090", cfg.Server.Port)
	}
	if !cfg.Database.Enabled {
		t.Fatal("database.enabled = false, want true")
	}
	if cfg.Database.URL == "" {
		t.Fatal("database.url is empty")
	}
	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Fatalf("server.read_timeout = %s, want 10s", cfg.Server.ReadTimeout)
	}
	if len(cfg.Server.AllowedOrigins) != 2 {
		t.Fatalf("allowed origins = %v, want 2 entries", cfg.Server.AllowedOrigins)
	}
}

func TestAllowedOriginsAreNormalized(t *testing.T) {
	// CORS and SSE compare origins separately, so spaces, trailing slashes and
	// duplicates must be removed once during configuration loading.
	t.Setenv(config.EnvAppConfigFile, filepath.Join("..", "..", "configs", "local.yaml.example"))
	t.Setenv(config.EnvCORSAllowedOrigins, " https://temukas.madgo.my.id/ , https://efa.madgo.my.id/ ,https://EFA.madgo.my.id , ")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []string{"https://temukas.madgo.my.id", "https://efa.madgo.my.id"}
	if len(cfg.Server.AllowedOrigins) != len(want) {
		t.Fatalf("allowed origins = %v, want %v", cfg.Server.AllowedOrigins, want)
	}
	for i := range want {
		if cfg.Server.AllowedOrigins[i] != want[i] {
			t.Fatalf("allowed origins = %v, want %v", cfg.Server.AllowedOrigins, want)
		}
	}
}
