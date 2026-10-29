package echohttp

import (
	"net/http"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/labstack/echo/v4"
)

func configHandler(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, s.Config())
	}
}

func state(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		response, err := s.GetState(c.Request().Context(), session.User)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, response)
	}
}

func suggestions(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		response, err := s.GetSuggestions(c.Request().Context(), session.User)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, response)
	}
}

func previewImport(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		var req models.ImportRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		if err := c.Validate(&req); err != nil {
			return err
		}
		response, err := s.PreviewImportForWorkspace(c.Request().Context(), session.User, req.Kind, req.CSV)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, response)
	}
}

func commitImport(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		var req models.ImportRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		if err := c.Validate(&req); err != nil {
			return err
		}
		response, err := s.CommitImport(c.Request().Context(), session.User, req)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, response)
	}
}

func cancelImport(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		if err := s.CancelImport(c.Request().Context(), session.User, c.Param("id")); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func allocate(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		var req models.AllocationRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		if err := c.Validate(&req); err != nil {
			return err
		}
		response, err := s.Allocate(c.Request().Context(), session.User, req)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, response)
	}
}

func undoAllocation(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		if err := s.UndoAllocation(c.Request().Context(), session.User, c.Param("id")); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func exportCSV(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		data, filename, err := s.ExportCSV(c.Request().Context(), session.User)
		if err != nil {
			return err
		}
		c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
		return c.Blob(http.StatusOK, "text/csv; charset=utf-8", data)
	}
}
