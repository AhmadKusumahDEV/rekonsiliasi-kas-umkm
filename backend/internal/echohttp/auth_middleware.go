package echohttp

import (
	"crypto/subtle"
	"net/http"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/config"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/models"
	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/internal/service"
	pkgErr "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/errors"
	"github.com/labstack/echo/v4"
)

func sessionMiddleware(s *service.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(config.SessionCookieName)
			if err != nil || cookie.Value == "" {
				return pkgErr.New(pkgErr.ErrUnauthorized, "Sesi berakhir. Masuk kembali untuk melanjutkan.")
			}

			session, err := s.GetSession(c.Request().Context(), cookie.Value)
			if err != nil {
				return err
			}
			if c.Request().Method != http.MethodGet && c.Request().Method != http.MethodHead {
				csrf := c.Request().Header.Get(config.HeaderCSRFToken)
				if subtle.ConstantTimeCompare([]byte(csrf), []byte(session.CSRF)) != 1 {
					return pkgErr.New(pkgErr.ErrForbidden, "Sesi formulir berubah. Muat ulang halaman sebelum menyimpan.")
				}
			}

			c.Set(config.ContextKeySession, session)
			return next(c)
		}
	}
}

func currentSession(c echo.Context) (models.Session, bool) {
	session, ok := c.Get(config.ContextKeySession).(models.Session)
	return session, ok
}

func sessionToken(c echo.Context) string {
	cookie, err := c.Cookie(config.SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
