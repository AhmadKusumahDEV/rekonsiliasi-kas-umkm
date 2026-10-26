package echohttp

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func securityHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		response := c.Response().Header()
		response.Set("X-Content-Type-Options", "nosniff")
		response.Set("X-Frame-Options", "DENY")
		response.Set("Referrer-Policy", "no-referrer")
		response.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		response.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		return next(c)
	}
}

func corsMiddleware(allowedOrigins []string) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			"X-Request-ID",
		},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60,
	})
}
