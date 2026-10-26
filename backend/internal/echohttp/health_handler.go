package echohttp

import (
	"net/http"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/labstack/echo/v4"
)

func liveness(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, s.Liveness())
	}
}

func readiness(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		response, healthy := s.Readiness(c.Request().Context())
		if !healthy {
			return c.JSON(http.StatusServiceUnavailable, response)
		}

		return c.JSON(http.StatusOK, response)
	}
}
