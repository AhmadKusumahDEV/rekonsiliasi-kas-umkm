package sse

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOriginDefaultMatchesScheme(t *testing.T) {
	cfg := Config{}
	req := httptest.NewRequest(http.MethodGet, "http://app.example/events", nil)
	req.Host = "app.example"
	req.Header.Set("Origin", "http://app.example")
	if !checkOrigin(cfg, req) {
		t.Fatal("same-host http origin was rejected")
	}
	req.Header.Set("Origin", "https://app.example")
	if checkOrigin(cfg, req) {
		t.Fatal("https origin was accepted on an http request")
	}
	req.TLS = &tls.ConnectionState{}
	req.Header.Set("Origin", "https://app.example")
	if !checkOrigin(cfg, req) {
		t.Fatal("same-host https origin was rejected")
	}
	req.Header.Set("Origin", "https://app.example/evil")
	if checkOrigin(cfg, req) {
		t.Fatal("origin with a path was accepted")
	}
	req.Header.Set("Origin", "https://user@app.example")
	if checkOrigin(cfg, req) {
		t.Fatal("origin with userinfo was accepted")
	}
}

func TestRejectCarriesCORSForValidOrigin(t *testing.T) {
	cfg := testConfig()
	cfg.AllowedOrigins = []string{"https://app.example"}
	cfg.Authenticate = func(*http.Request) (Identity, error) {
		return Identity{}, NewAuthError(http.StatusUnauthorized, "invalid token")
	}
	m := startTestManager(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://api.example/events", nil)
	req.Host = "api.example"
	req.Header.Set("Origin", "https://app.example")
	recorder := httptest.NewRecorder()
	m.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestRejectDoesNotEchoInvalidOrigin(t *testing.T) {
	cfg := testConfig()
	cfg.AllowedOrigins = []string{"https://app.example"}
	m := startTestManager(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://api.example/events", nil)
	req.Host = "api.example"
	req.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()
	m.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("invalid origin was echoed as %q", got)
	}
}
