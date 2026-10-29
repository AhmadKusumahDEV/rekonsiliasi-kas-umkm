package echohttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestSessionCookieThroughHTTPSProxy(t *testing.T) {
	for _, secure := range []bool{false, true} {
		e := echo.New()
		r := httptest.NewRequest(http.MethodGet, "http://temukas.example/", nil)
		// Echo trusts forwarded scheme headers only from a loopback/private
		// proxy address. Keep the test aligned with the real reverse-proxy path.
		r.RemoteAddr = "127.0.0.1:1234"
		if secure {
			r.Header.Set("X-Forwarded-Proto", "https")
		}
		w := httptest.NewRecorder()
		c := e.NewContext(r, w)
		setSessionCookie(c, "synthetic-session", time.Now().Add(time.Hour).Unix())
		cookie := w.Result().Cookies()[0]
		if cookie.Secure != secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
			t.Fatalf("unexpected cookie through proxy: %+v", cookie)
		}
		logout := httptest.NewRecorder()
		clearSessionCookie(e.NewContext(r, logout))
		if got := logout.Result().Cookies()[0]; got.Secure != secure || got.MaxAge != -1 {
			t.Fatalf("logout must expire the same secure cookie: %+v", got)
		}
	}
}

func TestCORSAllowsCSRFHeader(t *testing.T) {
	e := echo.New()
	e.Use(corsMiddleware([]string{"https://temukas.example"}))
	e.PUT("/api/v1/auth/password", func(c echo.Context) error { return c.NoContent(204) })
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/password", nil)
	r.Header.Set(echo.HeaderOrigin, "https://temukas.example")
	r.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPut)
	r.Header.Set(echo.HeaderAccessControlRequestHeaders, "content-type,x-csrf-token")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent || !strings.Contains(strings.ToLower(w.Header().Get(echo.HeaderAccessControlAllowHeaders)), "x-csrf-token") {
		t.Fatalf("CSRF header missing from preflight: %d %v", w.Code, w.Header())
	}
}

func TestCORSAllowsConfiguredFrontendOrigins(t *testing.T) {
	e := echo.New()
	e.Use(corsMiddleware([]string{"https://temukas.example", "https://efa.example"}))
	e.GET("/api/v1/config", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	for _, origin := range []string{"https://temukas.example", "https://efa.example"} {
		r := httptest.NewRequest(http.MethodOptions, "/api/v1/config", nil)
		r.Header.Set(echo.HeaderOrigin, origin)
		r.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
		w := httptest.NewRecorder()
		e.ServeHTTP(w, r)

		if w.Code != http.StatusNoContent {
			t.Fatalf("%s preflight status = %d", origin, w.Code)
		}
		if got := w.Header().Get(echo.HeaderAccessControlAllowOrigin); got != origin {
			t.Fatalf("%s allow-origin = %q", origin, got)
		}
		if got := w.Header().Get(echo.HeaderAccessControlAllowCredentials); got != "true" {
			t.Fatalf("%s allow-credentials = %q", origin, got)
		}
	}

	r := httptest.NewRequest(http.MethodOptions, "/api/v1/config", nil)
	r.Header.Set(echo.HeaderOrigin, "https://evil.example")
	r.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	if got := w.Header().Get(echo.HeaderAccessControlAllowOrigin); got != "" {
		t.Fatalf("unlisted origin got allow-origin %q", got)
	}
}

func TestAuthLimiterRestrictsRepeatedAttempts(t *testing.T) {
	e := echo.New()
	e.POST("/login", func(c echo.Context) error { return c.NoContent(204) }, authRateLimiter())
	for i := 0; i < 26; i++ {
		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		r.RemoteAddr = "192.0.2.1:1234"
		w := httptest.NewRecorder()
		e.ServeHTTP(w, r)
		want := 204
		if i == 25 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("attempt %d: got %d, want %d", i+1, w.Code, want)
		}
	}
}
