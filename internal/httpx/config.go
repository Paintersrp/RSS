package httpx

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	defaultHTTPAddr          = ":8080"
	defaultShutdownTimeout   = 5 * time.Second
	defaultDBMaxOpenConns    = 10
	defaultDBMaxIdleConns    = 10
	defaultDBConnMaxLifetime = 30 * time.Minute
	defaultDBPingTimeout     = 10 * time.Second
	defaultFetcherInterval   = 2 * time.Minute
	defaultFetcherBatchSize  = 250
	defaultBackoffMin        = 30 * time.Second
	defaultBackoffMax        = 10 * time.Minute
	defaultBackoffFactor     = 2.0
)

type RuntimeConfig struct {
	Service  string
	Database DatabaseConfig
	HTTP     HTTPConfig
	Search   SearchConfig
	Fetcher  FetcherConfig
	Expose   bool
}

type DatabaseConfig struct {
	Driver          string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	PingTimeout     time.Duration
}

type HTTPConfig struct {
	Addr            string
	ShutdownTimeout time.Duration
}

type SearchConfig struct {
	URL string
}

type FetcherConfig struct {
	Interval  time.Duration
	BatchSize int
	Backoff   BackoffConfig
}

type BackoffConfig struct {
	Min    time.Duration
	Max    time.Duration
	Factor float64
}

func LoadRuntimeConfig(service string) (RuntimeConfig, error) {
	cfg := RuntimeConfig{
		Service: service,
		Database: DatabaseConfig{
			Driver:          "pgx",
			MaxOpenConns:    defaultDBMaxOpenConns,
			MaxIdleConns:    defaultDBMaxIdleConns,
			ConnMaxLifetime: defaultDBConnMaxLifetime,
			PingTimeout:     defaultDBPingTimeout,
		},
		HTTP: HTTPConfig{
			Addr:            defaultHTTPAddr,
			ShutdownTimeout: defaultShutdownTimeout,
		},
		Fetcher: FetcherConfig{
			Interval:  defaultFetcherInterval,
			BatchSize: defaultFetcherBatchSize,
			Backoff: BackoffConfig{
				Min:    defaultBackoffMin,
				Max:    defaultBackoffMax,
				Factor: defaultBackoffFactor,
			},
		},
	}

	if v := envString("COURIER_SERVICE_NAME"); v != "" {
		cfg.Service = v
	}

	cfg.HTTP.Addr = stringWithDefault("COURIER_HTTP_ADDR", cfg.HTTP.Addr)

	shutdownTimeout, err := durationFromEnv("COURIER_HTTP_SHUTDOWN_TIMEOUT", cfg.HTTP.ShutdownTimeout)
	if err != nil {
		return cfg, err
	}
	if shutdownTimeout <= 0 {
		return cfg, fmt.Errorf("COURIER_HTTP_SHUTDOWN_TIMEOUT must be greater than zero")
	}
	cfg.HTTP.ShutdownTimeout = shutdownTimeout

	cfg.Database.Driver = stringWithDefault("COURIER_DB_DRIVER", cfg.Database.Driver)

	maxOpenConns, err := intFromEnv("COURIER_DB_MAX_OPEN_CONNS", cfg.Database.MaxOpenConns)
	if err != nil {
		return cfg, err
	}
	if maxOpenConns < 0 {
		return cfg, fmt.Errorf("COURIER_DB_MAX_OPEN_CONNS must be non-negative")
	}
	cfg.Database.MaxOpenConns = maxOpenConns

	maxIdleConns, err := intFromEnv("COURIER_DB_MAX_IDLE_CONNS", cfg.Database.MaxIdleConns)
	if err != nil {
		return cfg, err
	}
	if maxIdleConns < 0 {
		return cfg, fmt.Errorf("COURIER_DB_MAX_IDLE_CONNS must be non-negative")
	}
	cfg.Database.MaxIdleConns = maxIdleConns

	connMaxLifetime, err := durationFromEnv("COURIER_DB_CONN_MAX_LIFETIME", cfg.Database.ConnMaxLifetime)
	if err != nil {
		return cfg, err
	}
	cfg.Database.ConnMaxLifetime = connMaxLifetime

	pingTimeout, err := durationFromEnv("COURIER_DB_PING_TIMEOUT", cfg.Database.PingTimeout)
	if err != nil {
		return cfg, err
	}
	if pingTimeout <= 0 {
		return cfg, fmt.Errorf("COURIER_DB_PING_TIMEOUT must be greater than zero")
	}
	cfg.Database.PingTimeout = pingTimeout

	dsn := envString("COURIER_DSN")
	if dsn == "" {
		return cfg, fmt.Errorf("COURIER_DSN is required")
	}
	cfg.Database.DSN = dsn

	searchURL := envString("MEILI_URL")
	if searchURL == "" {
		return cfg, fmt.Errorf("MEILI_URL is required")
	}
	cfg.Search.URL = searchURL

	interval, err := durationFromEnv("COURIER_EVERY", cfg.Fetcher.Interval)
	if err != nil {
		return cfg, err
	}
	if interval <= 0 {
		return cfg, fmt.Errorf("COURIER_EVERY must be greater than zero")
	}
	cfg.Fetcher.Interval = interval

	batchSize, err := intFromEnv("COURIER_BATCH_UPSERT", cfg.Fetcher.BatchSize)
	if err != nil {
		return cfg, err
	}
	if batchSize <= 0 {
		return cfg, fmt.Errorf("COURIER_BATCH_UPSERT must be a positive integer")
	}
	cfg.Fetcher.BatchSize = batchSize

	backoffMin, err := durationFromEnv("COURIER_BACKOFF_MIN", cfg.Fetcher.Backoff.Min)
	if err != nil {
		return cfg, err
	}
	if backoffMin <= 0 {
		return cfg, fmt.Errorf("COURIER_BACKOFF_MIN must be greater than zero")
	}
	cfg.Fetcher.Backoff.Min = backoffMin

	backoffMax, err := durationFromEnv("COURIER_BACKOFF_MAX", cfg.Fetcher.Backoff.Max)
	if err != nil {
		return cfg, err
	}
	if backoffMax <= 0 {
		return cfg, fmt.Errorf("COURIER_BACKOFF_MAX must be greater than zero")
	}
	if backoffMax < cfg.Fetcher.Backoff.Min {
		return cfg, fmt.Errorf("COURIER_BACKOFF_MAX must be greater than or equal to COURIER_BACKOFF_MIN")
	}
	cfg.Fetcher.Backoff.Max = backoffMax

	backoffFactor, err := floatFromEnv("COURIER_BACKOFF_FACTOR", cfg.Fetcher.Backoff.Factor)
	if err != nil {
		return cfg, err
	}
	if backoffFactor <= 0 {
		return cfg, fmt.Errorf("COURIER_BACKOFF_FACTOR must be greater than zero")
	}
	cfg.Fetcher.Backoff.Factor = backoffFactor

	if v := envString("COURIER_EXPOSE_CONFIG"); v != "" {
		expose, err := strconv.ParseBool(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid COURIER_EXPOSE_CONFIG: %w", err)
		}
		cfg.Expose = expose
	}

	return cfg, nil
}

type RuntimeConfigSnapshot struct {
	Service  string           `json:"service"`
	HTTP     HTTPSnapshot     `json:"http"`
	Database DatabaseSnapshot `json:"database"`
	Search   SearchSnapshot   `json:"search"`
	Fetcher  FetcherSnapshot  `json:"fetcher"`
}

type HTTPSnapshot struct {
	Addr            string `json:"addr"`
	ShutdownTimeout string `json:"shutdown_timeout"`
}

type DatabaseSnapshot struct {
	Driver          string `json:"driver"`
	DSN             string `json:"dsn"`
	MaxOpenConns    int    `json:"max_open_conns"`
	MaxIdleConns    int    `json:"max_idle_conns"`
	ConnMaxLifetime string `json:"conn_max_lifetime"`
	PingTimeout     string `json:"ping_timeout"`
}

type SearchSnapshot struct {
	URL string `json:"url"`
}

type FetcherSnapshot struct {
	Interval  string          `json:"interval"`
	BatchSize int             `json:"batch_size"`
	Backoff   BackoffSnapshot `json:"backoff"`
}

type BackoffSnapshot struct {
	Min    string  `json:"min"`
	Max    string  `json:"max"`
	Factor float64 `json:"factor"`
}

func (cfg RuntimeConfig) Snapshot() RuntimeConfigSnapshot {
	return RuntimeConfigSnapshot{
		Service: cfg.Service,
		HTTP: HTTPSnapshot{
			Addr:            cfg.HTTP.Addr,
			ShutdownTimeout: cfg.HTTP.ShutdownTimeout.String(),
		},
		Database: DatabaseSnapshot{
			Driver:          cfg.Database.Driver,
			DSN:             sanitizeDSN(cfg.Database.DSN),
			MaxOpenConns:    cfg.Database.MaxOpenConns,
			MaxIdleConns:    cfg.Database.MaxIdleConns,
			ConnMaxLifetime: cfg.Database.ConnMaxLifetime.String(),
			PingTimeout:     cfg.Database.PingTimeout.String(),
		},
		Search: SearchSnapshot{
			URL: cfg.Search.URL,
		},
		Fetcher: FetcherSnapshot{
			Interval:  cfg.Fetcher.Interval.String(),
			BatchSize: cfg.Fetcher.BatchSize,
			Backoff: BackoffSnapshot{
				Min:    cfg.Fetcher.Backoff.Min.String(),
				Max:    cfg.Fetcher.Backoff.Max.String(),
				Factor: cfg.Fetcher.Backoff.Factor,
			},
		},
	}
}

func RegisterConfigRoute(e *echo.Echo, cfg RuntimeConfig) {
	if !cfg.Expose {
		return
	}

	e.GET("/config", func(c echo.Context) error {
		return c.JSON(http.StatusOK, cfg.Snapshot())
	})
}

func sanitizeDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "<redacted>"
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if username != "" {
			parsed.User = url.User(username)
		} else {
			parsed.User = url.User("")
		}
	}
	return parsed.String()
}

func envString(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func stringWithDefault(key, fallback string) string {
	if v := envString(key); v != "" {
		return v
	}
	return fallback
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	if v := envString(key); v != "" {
		duration, err := time.ParseDuration(v)
		if err != nil {
			return 0, fmt.Errorf("invalid %s: %w", key, err)
		}
		if duration < 0 {
			return 0, fmt.Errorf("%s must not be negative", key)
		}
		return duration, nil
	}
	return fallback, nil
}

func intFromEnv(key string, fallback int) (int, error) {
	if v := envString(key); v != "" {
		value, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("invalid %s: %w", key, err)
		}
		return value, nil
	}
	return fallback, nil
}

func floatFromEnv(key string, fallback float64) (float64, error) {
	if v := envString(key); v != "" {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid %s: %w", key, err)
		}
		return value, nil
	}
	return fallback, nil
}
