// seed-demo creates synthetic workspaces explicitly; it never resets existing users.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/database"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if os.Getenv("ALLOW_DEMO_SEED") != "true" {
		return errors.New("set ALLOW_DEMO_SEED=true only for a disposable demo database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !cfg.Database.Enabled {
		return errors.New("enable database and apply migrations first")
	}
	db, err := database.New(ctx, cfg.Database.ToDatabaseConfig())
	if err != nil {
		return err
	}
	defer db.Close()
	svc := service.NewService(cfg, db, nil)
	accounts := []models.RegisterRequest{
		{Name: "Evan William", Email: "pemilik@temukas.test", Password: "KasDemo!2026", Workspace: "Roti Sore (Demo)", Role: "owner"},
		{Name: "Pemilik Kedua", Email: "pemilik.dua@temukas.test", Password: "UsahaDemo!2026", Workspace: "Kedai Pagi (Demo)", Role: "owner"},
	}
	for _, account := range accounts {
		auth, registerErr := svc.Register(ctx, account)
		if errors.Is(registerErr, pkgErr.ErrConflict) {
			fmt.Printf("Skipped existing account: %s (no password or data changed)\n", account.Email)
			continue
		}
		if registerErr != nil {
			return fmt.Errorf("register %s: %w", account.Email, registerErr)
		}
		for _, item := range []struct{ kind, file string }{
			{"sale", "contoh-penjualan.csv"}, {"receipt", "contoh-penerimaan.csv"},
		} {
			csv, readErr := os.ReadFile(filepath.Join("..", "frontend", "assets", item.file))
			if readErr != nil {
				return readErr
			}
			if _, err = svc.CommitImport(ctx, auth.User, models.ImportRequest{Kind: item.kind, CSV: string(csv)}); err != nil {
				return fmt.Errorf("import %s: %w", item.file, err)
			}
		}
		if err = svc.Logout(ctx, auth.Token); err != nil {
			return err
		}
		fmt.Printf("Created synthetic demo workspace: %s\n", account.Email)
	}
	return nil
}
