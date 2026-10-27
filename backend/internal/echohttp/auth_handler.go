package echohttp

import (
	"net/http"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	"github.com/labstack/echo/v4"
)

func register(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req models.RegisterRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		if err := c.Validate(&req); err != nil {
			return err
		}

		result, err := s.Register(c.Request().Context(), req)
		if err != nil {
			return err
		}
		setSessionCookie(c, result.Token, result.ExpiresAt)
		return c.JSON(http.StatusCreated, models.AuthResponse{User: result.User, CSRF: result.CSRF})
	}
}

func login(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req models.LoginRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		if err := c.Validate(&req); err != nil {
			return err
		}

		result, err := s.Login(c.Request().Context(), req)
		if err != nil {
			return err
		}
		setSessionCookie(c, result.Token, result.ExpiresAt)
		return c.JSON(http.StatusOK, models.AuthResponse{User: result.User, CSRF: result.CSRF})
	}
}

func me(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		return c.JSON(http.StatusOK, models.AuthResponse{User: session.User, CSRF: session.CSRF})
	}
}

func logout(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, _ := currentSession(c)
		if err := s.Logout(c.Request().Context(), sessionToken(c)); err != nil {
			return err
		}
		s.DisconnectUser(session.User.ID)
		clearSessionCookie(c)
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func changePassword(s *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		session, ok := currentSession(c)
		if !ok {
			return echo.ErrUnauthorized
		}
		var req models.ChangePasswordRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		if err := c.Validate(&req); err != nil {
			return err
		}
		if err := s.ChangePassword(c.Request().Context(), session.User, req); err != nil {
			return err
		}
		s.DisconnectUser(session.User.ID)
		clearSessionCookie(c)
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func setSessionCookie(c echo.Context, token string, expiresAt int64) {
	c.SetCookie(&http.Cookie{
		Name:     config.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(expiresAt, 0),
		MaxAge:   int(config.SessionTTL.Seconds()),
	})
}

func clearSessionCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     config.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
