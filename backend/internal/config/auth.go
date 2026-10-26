package config

import "time"

const (
	AppKindUMKM       = "umkm"
	SessionCookieName = "rku_session"
	SessionTTL        = 12 * time.Hour
	ContextKeySession = "rku_session"
	APIPrefix         = "/api/v1"
)
