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

	dsn := strings.TrimSpace(os.Getenv("COURIER_DSN"))
	if dsn == "" {
		return cfg, fmt.Errorf("COURIER_DSN is required")
	}
	cfg.Database.DSN = dsn

	searchURL := strings.TrimSpace(os.Getenv("MEILI_URL"))
	if searchURL == "" {
		return cfg, fmt.Errorf("MEILI_URL is required")
	}
	cfg.Search.URL = searchURL

	if v := strings.TrimSpace(os.Getenv("COURIER_EVERY")); v != "" {
		interval, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid COURIER_EVERY: %w", err)
		}
		cfg.Fetcher.Interval = interval
	}

	if v := strings.TrimSpace(os.Getenv("COURIER_BATCH_UPSERT")); v != "" {
		batchSize, err := strconv.Atoi(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid COURIER_BATCH_UPSERT: %w", err)
		}
		if batchSize <= 0 {
			return cfg, fmt.Errorf("COURIER_BATCH_UPSERT must be a positive integer")
		}
		cfg.Fetcher.BatchSize = batchSize
	}

	if v := strings.TrimSpace(os.Getenv("COURIER_EXPOSE_CONFIG")); v != "" {
		expose, err := strconv.ParseBool(v)
		if err == nil {
			cfg.Expose = expose
		}
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
