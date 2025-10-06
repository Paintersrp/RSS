package main

import (
	"context"
	"database/sql"
	"log"
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
		log.Fatal("COURIER_DSN is required")
	}
	meiliURL := os.Getenv("MEILI_URL")
	if meiliURL == "" {
		log.Fatal("MEILI_URL is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	searchClient := search.New(meiliURL)
	if err := searchClient.EnsureIndex(ctx); err != nil {
		log.Fatalf("ensure index: %v", err)
	}

	store := store.New(db)
	srv := httpx.NewServer(httpx.Config{
		Store:   store,
		Search:  searchClient,
		DB:      db,
		Service: svc,
	})

	go func() {
		logx.Info(svc, "listening", map[string]any{"addr": ":8080"})
		if err := srv.Start(":8080"); err != nil {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logx.Error(svc, "shutdown", err, nil)
	}
}
