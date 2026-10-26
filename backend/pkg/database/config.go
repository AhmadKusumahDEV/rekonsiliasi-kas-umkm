package database

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config describes a PostgreSQL pool. URL takes precedence when it is set;
// otherwise the individual connection fields are combined into a DSN.
type Config struct {
	URL               string
	Host              string
	Port              int
	Username          string
	Password          string
	Database          string
	SSLMode           string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout    time.Duration
}

func (c Config) withDefaults() Config {
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.MaxConns == 0 {
		c.MaxConns = 10
	}
	if c.MinConns < 0 {
		c.MinConns = 0
	}
	if c.MaxConnLifetime == 0 {
		c.MaxConnLifetime = 30 * time.Minute
	}
	if c.MaxConnIdleTime == 0 {
		c.MaxConnIdleTime = 5 * time.Minute
	}
	if c.HealthCheckPeriod == 0 {
		c.HealthCheckPeriod = 30 * time.Second
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 10 * time.Second
	}

	return c
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.URL) != "" {
		parsed, err := url.Parse(c.URL)
		if err != nil {
			return fmt.Errorf("database: invalid url: %w", err)
		}
		if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
			return fmt.Errorf("database: url scheme must be postgres or postgresql")
		}
		if parsed.Host == "" {
			return fmt.Errorf("database: url host is empty")
		}
	} else {
		if strings.TrimSpace(c.Host) == "" {
			return fmt.Errorf("database: host is empty")
		}
		if strings.TrimSpace(c.Database) == "" {
			return fmt.Errorf("database: name is empty")
		}
	}

	if c.Port <= 0 {
		return fmt.Errorf("database: port must be positive")
	}
	if c.MaxConns < c.MinConns {
		return fmt.Errorf("database: max conns must be greater than or equal to min conns")
	}

	return nil
}

func (c Config) DSN() string {
	if strings.TrimSpace(c.URL) != "" {
		return c.URL
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.Username, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   c.Database,
	}
	query := u.Query()
	query.Set("sslmode", c.SSLMode)
	u.RawQuery = query.Encode()

	return u.String()
}

// ConfigProvider keeps pkg/database independent from internal/config. It is
// useful when another entrypoint has its own configuration source.
type ConfigProvider interface {
	GetString(key string) string
	GetInt(key string) int
	GetDuration(key string) time.Duration
}

func ConfigFrom(provider ConfigProvider, prefix string) Config {
	return Config{
		Host:              provider.GetString(prefix + ".host"),
		Port:              provider.GetInt(prefix + ".port"),
		Username:          provider.GetString(prefix + ".username"),
		Password:          provider.GetString(prefix + ".password"),
		Database:          provider.GetString(prefix + ".name"),
		SSLMode:           provider.GetString(prefix + ".sslmode"),
		MaxConns:          int32(provider.GetInt(prefix + ".max_conns")),
		MinConns:          int32(provider.GetInt(prefix + ".min_conns")),
		MaxConnLifetime:   provider.GetDuration(prefix + ".max_conn_lifetime"),
		MaxConnIdleTime:   provider.GetDuration(prefix + ".max_conn_idle_time"),
		HealthCheckPeriod: provider.GetDuration(prefix + ".health_check_period"),
		ConnectTimeout:    provider.GetDuration(prefix + ".connect_timeout"),
	}.withDefaults()
}
