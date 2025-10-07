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
	dsn := os.Getenv("COURIER_DSN")
	if dsn == "" {
		fatal(svc, "missing required env var", errors.New("COURIER_DSN is required"), map[string]any{"env": "COURIER_DSN"})
	}
	meiliURL := os.Getenv("MEILI_URL")
	if meiliURL == "" {
		fatal(svc, "missing required env var", errors.New("MEILI_URL is required"), map[string]any{"env": "MEILI_URL"})
	}

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

	searchClient := search.New(meiliURL)
	if err := searchClient.EnsureIndex(ctx); err != nil {
		fatal(svc, "ensure index", err, nil)
	}

	store := store.New(db)
	srv := httpx.NewServer(httpx.Config{
		Store:   store,
		Search:  searchClient,
		DB:      db,
		Service: svc,
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
		logx.Error(svc, "server", err, map[string]any{"addr": addr})
		os.Exit(1)
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
		logx.Error(svc, "server", err, map[string]any{"addr": addr})
		os.Exit(1)
	}
}

func fatal(service, msg string, err error, extra map[string]any) {
	logx.Error(service, msg, err, extra)
	os.Exit(1)
}
