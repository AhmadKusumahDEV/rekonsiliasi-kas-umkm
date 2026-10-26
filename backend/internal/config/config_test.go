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
