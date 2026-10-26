// Package service contains application use cases and business rules. It does
// not import Echo or any other HTTP transport package.
package service

import (
	"context"
	"net/http"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/database"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/utils"
)

// Version is set at build time with -ldflags.
var Version = "dev"

type Service struct {
	cfg        config.Config
	db         *database.DB
	httpClient *http.Client
	startedAt  time.Time
}

func NewService(cfg config.Config, db *database.DB, httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &Service{
		cfg:        cfg,
		db:         db,
		httpClient: httpClient,
		startedAt:  time.Now(),
	}
}

func (s *Service) Liveness() models.LivenessResponse {
	return models.LivenessResponse{
		Status:        "ok",
		Service:       s.cfg.App.Name,
		Version:       Version,
		UptimeSeconds: int64(time.Since(s.startedAt).Seconds()),
		Time:          utils.NowJakarta().Format(time.RFC3339),
	}
}

// Readiness reports dependency health without exposing connection strings or
// driver errors. A disabled database is reported explicitly during local
// scaffolding, while the endpoint still answers because no domain endpoint is
// registered yet.
func (s *Service) Readiness(ctx context.Context) (models.ReadinessResponse, bool) {
	checks := map[string]models.HealthCheck{}
	status := "ok"
	healthy := true

	if s.db == nil {
		checkStatus := "disabled"
		if s.cfg.Database.Enabled {
			checkStatus = "error"
			status = "unavailable"
			healthy = false
		}
		checks["postgres"] = models.HealthCheck{Status: checkStatus, Message: "database belum terhubung"}
	} else {
		started := time.Now()
		err := s.db.Health(ctx)
		result := models.HealthCheck{
			Status:    "ok",
			LatencyMS: time.Since(started).Milliseconds(),
		}
		if err != nil {
			result.Status = "error"
			result.Message = "database tidak dapat dijangkau"
			status = "unavailable"
			healthy = false
		}
		checks["postgres"] = result
	}

	return models.ReadinessResponse{
		Status:  status,
		Service: s.cfg.App.Name,
		Version: Version,
		Checks:  checks,
	}, healthy
}
