package echohttp

import (
	"net/http"
	"time"

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
			"X-CSRF-Token",
		},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60,
	})
}

// The production API listens only behind the trusted reverse proxy, which
// overwrites forwarded client headers. Direct local clients use their peer IP.
func authRateLimiter() echo.MiddlewareFunc {
	store := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate: 25.0 / 600, Burst: 25, ExpiresIn: 10 * time.Minute,
	})
	return middleware.RateLimiter(store)
}
