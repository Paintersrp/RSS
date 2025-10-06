package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"courier/internal/feed"
	"courier/internal/item"
	"courier/internal/logx"
	"courier/internal/search"
	"courier/internal/store"
)

func main() {
	svc := "fetcher"
	dsn := os.Getenv("COURIER_DSN")
	if dsn == "" {
		log.Fatal("COURIER_DSN is required")
	}
	meiliURL := os.Getenv("MEILI_URL")
	if meiliURL == "" {
		log.Fatal("MEILI_URL is required")
	}
	every := 2 * time.Minute
	if v := os.Getenv("COURIER_EVERY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			every = d
		}
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := store.New(db)
	searchClient := search.New(meiliURL)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	if err := searchClient.EnsureIndex(ctx); err != nil {
		log.Fatalf("ensure index: %v", err)
	}
	cancel()

	fetcher := feed.NewFetcher()

	logx.Info(svc, "ready", map[string]any{"every": every.String()})

	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), every)
		run(ctx, svc, repo, searchClient, fetcher)
		cancel()
		<-ticker.C
	}
}

func run(ctx context.Context, svc string, repo *store.Store, searchClient *search.Client, fetcher *feed.Fetcher) {
	feeds, err := repo.ListFeeds(ctx, true)
	if err != nil {
		logx.Error(svc, "list feeds", err, nil)
		return
	}
	for _, f := range feeds {
		crawl(ctx, svc, repo, searchClient, fetcher, f)
	}
}

func crawl(ctx context.Context, svc string, repo *store.Store, searchClient *search.Client, fetcher *feed.Fetcher, f store.Feed) {
	etag := ""
	if f.ETag.Valid {
		etag = f.ETag.String
	}
	lastModified := ""
	if f.LastModified.Valid {
		lastModified = f.LastModified.String
	}

	res, err := fetcher.Fetch(ctx, f.URL, etag, lastModified)
	if err != nil {
		logx.Error(svc, "fetch", err, map[string]any{"feed": f.URL})
		return
	}

	now := time.Now().UTC()
	title := f.Title
	if res.Feed != nil && res.Feed.Title != "" {
		title = res.Feed.Title
	}

	_, err = repo.UpdateFeedCrawlState(ctx, store.UpdateFeedCrawlStateParams{
		ID:           f.ID,
		ETag:         sqlNullString(res.ETag),
		LastModified: sqlNullString(res.LastModified),
		LastCrawled:  sqlNullTime(now),
		Title:        title,
	})
	if err != nil {
		logx.Error(svc, "update feed", err, map[string]any{"feed": f.URL})
	}

	if res.Status == http.StatusNotModified || res.Feed == nil {
		logx.Info(svc, "feed not modified", map[string]any{"feed": f.URL})
		return
	}

	var docs []search.Document
	for _, entry := range res.Feed.Items {
		params := item.FromFeedItem(f.ID, entry)
		result, err := repo.UpsertItem(ctx, params)
		if err != nil {
			logx.Error(svc, "upsert item", err, map[string]any{"feed": f.URL})
			continue
		}
		doc := search.Document{
			ID:          result.Item.ID,
			FeedID:      result.Item.FeedID,
			FeedTitle:   result.Item.FeedTitle,
			Title:       result.Item.Title,
			ContentText: result.Item.ContentText,
			URL:         result.Item.URL,
		}
		if result.Item.PublishedAt.Valid {
			t := result.Item.PublishedAt.Time.UTC()
			doc.PublishedAt = &t
		}
		docs = append(docs, doc)
		if result.Fresh {
			logx.Info(svc, "item indexed", map[string]any{"id": result.Item.ID, "feed": f.URL})
		}
	}

	if err := searchClient.UpsertDocuments(ctx, docs); err != nil {
		logx.Error(svc, "search upsert", err, map[string]any{"feed": f.URL, "count": len(docs)})
	} else {
		logx.Info(svc, "feed processed", map[string]any{"feed": f.URL, "items": len(docs)})
	}
}

func sqlNullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: v}
}

func sqlNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Valid: true, Time: t}
}
