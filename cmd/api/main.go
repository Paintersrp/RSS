package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"

	"courier/internal/httpx"
	"courier/internal/logx"
	"courier/internal/search"
	"courier/internal/store"
)

func main() {
	svc := "api"

	runtimeCfg, err := httpx.LoadRuntimeConfig(svc)
	if err != nil {
		fatal(svc, "load config", err, nil)
	}
	svc = runtimeCfg.Service

	metrics := httpx.NewMetrics(svc)

	db, err := sql.Open(runtimeCfg.Database.Driver, runtimeCfg.Database.DSN)
	if err != nil {
		fatal(svc, "open db", err, nil)
	}
	defer db.Close()
	db.SetMaxOpenConns(runtimeCfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(runtimeCfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(runtimeCfg.Database.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), runtimeCfg.Database.PingTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		fatal(svc, "ping db", err, nil)
	}

	searchClient := search.New(runtimeCfg.Search.URL, metrics)
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
	srv.HTTPErrorHandler = httpx.HTTPErrorHandler(svc)
	httpx.RegisterConfigRoute(srv, runtimeCfg)

	addr := runtimeCfg.HTTP.Addr

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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), runtimeCfg.HTTP.ShutdownTimeout)
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
