// Package config loads application settings from configs/local.yaml and the
// environment. Viper is used only during bootstrap; the rest of the application
// receives a typed Config value.
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/database"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
}

type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	AllowedOrigins  []string      `mapstructure:"allowed_origins"`
}

func (s ServerConfig) Address() string {
	return net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))
}

type DatabaseConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	URL               string        `mapstructure:"url"`
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	Username          string        `mapstructure:"username"`
	Password          string        `mapstructure:"password"`
	Name              string        `mapstructure:"name"`
	SSLMode           string        `mapstructure:"sslmode"`
	MaxConns          int32         `mapstructure:"max_conns"`
	MinConns          int32         `mapstructure:"min_conns"`
	MaxConnLifetime   time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period"`
	ConnectTimeout    time.Duration `mapstructure:"connect_timeout"`
}

func (d DatabaseConfig) ToDatabaseConfig() database.Config {
	return database.Config{
		URL:               d.URL,
		Host:              d.Host,
		Port:              d.Port,
		Username:          d.Username,
		Password:          d.Password,
		Database:          d.Name,
		SSLMode:           d.SSLMode,
		MaxConns:          d.MaxConns,
		MinConns:          d.MinConns,
		MaxConnLifetime:   d.MaxConnLifetime,
		MaxConnIdleTime:   d.MaxConnIdleTime,
		HealthCheckPeriod: d.HealthCheckPeriod,
		ConnectTimeout:    d.ConnectTimeout,
	}
}

// Load reads configs/local.yaml when it exists. Environment variables can
// override the same keys, which keeps container configuration out of the image.
func Load() (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("local")
	v.AddConfigPath("configs")
	v.AddConfigPath("../configs")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if path := strings.TrimSpace(os.Getenv(EnvAppConfigFile)); path != "" {
		v.SetConfigFile(path)
	}

	for key, env := range map[string]string{
		KeyAppName:         EnvAppName,
		KeyAppEnvironment:  EnvAppEnvironment,
		KeyServerHost:      EnvServerHost,
		KeyServerPort:      EnvServerPort,
		KeyDatabaseEnabled: EnvDatabaseEnabled,
		KeyDatabaseURL:     EnvDatabaseURL,
		KeyDatabaseHost:    EnvDatabaseHost,
		KeyDatabasePort:    EnvDatabasePort,
		KeyDatabaseUser:    EnvDatabaseUser,
		KeyDatabasePass:    EnvDatabasePassword,
		KeyDatabaseName:    EnvDatabaseName,
		KeyDatabaseSSLMode: EnvDatabaseSSLMode,
	} {
		if err := v.BindEnv(key, env); err != nil {
			return Config{}, fmt.Errorf("bind config %s: %w", key, err)
		}
	}

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	cfg.applyDefaults()
	if origins := strings.TrimSpace(os.Getenv(EnvCORSAllowedOrigins)); origins != "" {
		cfg.Server.AllowedOrigins = splitCSV(origins)
	}
	cfg.Server.AllowedOrigins = normalizeOrigins(cfg.Server.AllowedOrigins)
	if len(cfg.Server.AllowedOrigins) == 0 {
		cfg.Server.AllowedOrigins = []string{"http://localhost:3000", "http://localhost:5173"}
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault(KeyAppName, "rekonsiliasi-kas-umkm")
	v.SetDefault(KeyAppEnvironment, "development")
	v.SetDefault(KeyServerHost, "0.0.0.0")
	v.SetDefault(KeyServerPort, 8080)
	v.SetDefault(KeyServerReadTimeout, "10s")
	v.SetDefault(KeyServerWriteTimeout, "15s")
	v.SetDefault(KeyServerIdleTimeout, "60s")
	v.SetDefault(KeyServerShutdownTimeout, "10s")
	v.SetDefault(KeyServerAllowedOrigins, []string{"http://localhost:3000", "http://localhost:5173"})
	v.SetDefault(KeyDatabaseEnabled, false)
	v.SetDefault(KeyDatabaseHost, "localhost")
	v.SetDefault(KeyDatabasePort, 5432)
	v.SetDefault(KeyDatabaseUser, "postgres")
	v.SetDefault(KeyDatabaseName, "rekonsiliasi_kas_umkm")
	v.SetDefault(KeyDatabaseSSLMode, "disable")
	v.SetDefault(KeyDatabaseMaxConns, 10)
	v.SetDefault(KeyDatabaseMinConns, 0)
	v.SetDefault(KeyDatabaseMaxConnLifetime, "30m")
	v.SetDefault(KeyDatabaseMaxConnIdleTime, "5m")
	v.SetDefault(KeyDatabaseHealthCheckPeriod, "30s")
	v.SetDefault(KeyDatabaseConnectTimeout, "10s")
}

func (c *Config) applyDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 10 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 15 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 60 * time.Second
	}
	if c.Server.ShutdownTimeout == 0 {
		c.Server.ShutdownTimeout = 10 * time.Second
	}
	if len(c.Server.AllowedOrigins) == 0 {
		c.Server.AllowedOrigins = []string{"http://localhost:3000", "http://localhost:5173"}
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.MaxConns == 0 {
		c.Database.MaxConns = 10
	}
	if c.Database.MaxConnLifetime == 0 {
		c.Database.MaxConnLifetime = 30 * time.Minute
	}
	if c.Database.MaxConnIdleTime == 0 {
		c.Database.MaxConnIdleTime = 5 * time.Minute
	}
	if c.Database.HealthCheckPeriod == 0 {
		c.Database.HealthCheckPeriod = 30 * time.Second
	}
	if c.Database.ConnectTimeout == 0 {
		c.Database.ConnectTimeout = 10 * time.Second
	}
}

func (c Config) validate() error {
	if strings.TrimSpace(c.App.Name) == "" {
		return fmt.Errorf("app.name is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Server.ShutdownTimeout <= 0 {
		return fmt.Errorf("server.shutdown_timeout must be positive")
	}

	return nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// normalizeOrigins keeps CORS and SSE using the same canonical values. A
// trailing slash or duplicate entry would otherwise make Echo accept an origin
// that the SSE handshake still rejects.
func normalizeOrigins(origins []string) []string {
	seen := make(map[string]struct{}, len(origins))
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin != "*" {
			origin = strings.TrimRight(origin, "/")
		}
		key := strings.ToLower(origin)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, origin)
	}
	return result
}
