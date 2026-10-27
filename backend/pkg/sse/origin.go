package sse

import (
	"net/http"
	"net/url"
	"strings"
)

func checkOrigin(cfg Config, r *http.Request) bool {
	if cfg.CheckOrigin != nil {
		return cfg.CheckOrigin(r)
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if origin == "null" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	if parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	if len(cfg.AllowedOrigins) == 0 {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		return strings.EqualFold(parsed.Scheme, scheme) && strings.EqualFold(parsed.Host, r.Host)
	}
	normalized := strings.TrimSuffix(origin, "/")
	for _, allowed := range cfg.AllowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" || strings.EqualFold(strings.TrimSuffix(allowed, "/"), normalized) {
			return true
		}
	}
	return false
}

func addCORSHeaders(cfg Config, r *http.Request, header http.Header) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return
	}
	if cfg.AllowCredentials {
		header.Set("Access-Control-Allow-Origin", origin)
		header.Set("Access-Control-Allow-Credentials", "true")
	} else if containsOrigin(cfg.AllowedOrigins, "*") {
		header.Set("Access-Control-Allow-Origin", "*")
	} else {
		header.Set("Access-Control-Allow-Origin", origin)
	}
	header.Add("Vary", "Origin")
	header.Set("Access-Control-Expose-Headers", "Last-Event-ID")
}

func containsOrigin(origins []string, wanted string) bool {
	for _, origin := range origins {
		if strings.TrimSpace(origin) == wanted {
			return true
		}
	}
	return false
}
