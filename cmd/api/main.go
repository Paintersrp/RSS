package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"courier/internal/httpx"
	"courier/internal/logx"
	"courier/internal/search"
	"courier/internal/store"
)

func main() {
	svc := "api"
	dsn := requireEnv(svc, "COURIER_DSN")
	meiliURL := requireEnv(svc, "MEILI_URL")

	metrics := httpx.NewMetrics(svc)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fatal(svc, "open db", err, nil)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		fatal(svc, "ping db", err, nil)
	}

	searchClient := search.New(meiliURL, metrics)
	if err := searchClient.EnsureIndex(ctx); err != nil {
		fatal(svc, "ensure index", err, nil)
	}

	store := store.New(db, metrics)
	srv := httpx.NewServer(httpx.Config{
		Store:   store,
		Search:  searchClient,
		DB:      db,
		Service: svc,
		Metrics: metrics,
	})

	const addr = ":8080"

	serverErrCh := make(chan error, 1)
	go func() {
		logx.Info(svc, "listening", map[string]any{"addr": addr})
		serverErrCh <- srv.Start(addr)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stop:
	case err := <-serverErrCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			logx.Info(svc, "server stopped", map[string]any{"addr": addr})
			return
		}
		fatal(svc, "server", err, map[string]any{"addr": addr})
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logx.Error(svc, "shutdown", err, nil)
	}

	if err := <-serverErrCh; err == nil || errors.Is(err, http.ErrServerClosed) {
		logx.Info(svc, "server stopped", map[string]any{"addr": addr})
		return
	} else {
		fatal(svc, "server", err, map[string]any{"addr": addr})
	}
}

func fatal(service, msg string, err error, extra map[string]any) {
	logx.Error(service, msg, err, extra)
	os.Exit(1)
}

func requireEnv(service, key string) string {
	value := os.Getenv(key)
	if value == "" {
		fatal(service, "missing required env var", errors.New(key+" is required"), map[string]any{"env": key})
	}
	return value
}
