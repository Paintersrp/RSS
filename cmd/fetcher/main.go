package main

import (
	"context"
	"database/sql"
	"errors"
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
	dsn := requireEnv(svc, "COURIER_DSN")
	meiliURL := requireEnv(svc, "MEILI_URL")
	every := 2 * time.Minute
	if v := os.Getenv("COURIER_EVERY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			every = d
		}
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fatal(svc, "open db", err, nil)
	}
	defer db.Close()

	repo := store.New(db)
	searchClient := search.New(meiliURL)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := db.PingContext(ctx); err != nil {
		fatal(svc, "ping db", err, nil)
	}
	if err := searchClient.EnsureIndex(ctx); err != nil {
		fatal(svc, "ensure index", err, nil)
	}
	cancel()

	fetcher := feed.NewFetcher()
	backoffs := newBackoffTracker()

	logx.Info(svc, "ready", map[string]any{"every": every.String()})

	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), every)
		run(ctx, svc, repo, searchClient, fetcher, backoffs)
		cancel()
		<-ticker.C
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

func run(ctx context.Context, svc string, repo *store.Store, searchClient *search.Client, fetcher *feed.Fetcher, backoffs *backoffTracker) {
	feeds, err := repo.ListFeeds(ctx, true)
	if err != nil {
		logx.Error(svc, "list feeds", err, nil)
		return
	}
	now := time.Now().UTC()
	for _, f := range feeds {
		if wait := backoffs.Remaining(f.ID, now); wait > 0 {
			logx.Info(svc, "backoff active", map[string]any{"feed": f.URL, "retry_in": wait.String()})
			continue
		}
		crawl(ctx, svc, repo, searchClient, fetcher, backoffs, f)
	}
}

func crawl(ctx context.Context, svc string, repo *store.Store, searchClient *search.Client, fetcher *feed.Fetcher, backoffs *backoffTracker, f store.Feed) {
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
		now := time.Now().UTC()
		if errors.Is(err, feed.ErrRetryLater) {
			duration := backoffs.Schedule(f.ID, now, res.RetryAfter)
			logx.Error(svc, "fetch retry", err, map[string]any{"feed": f.URL, "status": res.Status, "retry_in": duration.String()})
			return
		}
		duration := backoffs.Schedule(f.ID, now, 0)
		logx.Error(svc, "fetch", err, map[string]any{"feed": f.URL, "retry_in": duration.String()})
		return
	}

	backoffs.Reset(f.ID)

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
		batches := 0
		if size := searchClient.BatchSize(); size > 0 {
			batches = (len(docs) + size - 1) / size
		}
		logx.Info(svc, "feed processed", map[string]any{"feed": f.URL, "items": len(docs), "batches": batches})
	}
}

func sqlNullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: v}
}

type backoffTracker struct {
	min    time.Duration
	max    time.Duration
	factor float64
	items  map[string]backoffEntry
}

type backoffEntry struct {
	until    time.Time
	duration time.Duration
}

func newBackoffTracker() *backoffTracker {
	return &backoffTracker{
		min:    30 * time.Second,
		max:    10 * time.Minute,
		factor: 2.0,
		items:  make(map[string]backoffEntry),
	}
}

func (b *backoffTracker) Remaining(id string, now time.Time) time.Duration {
	entry, ok := b.items[id]
	if !ok {
		return 0
	}
	if now.After(entry.until) {
		delete(b.items, id)
		return 0
	}
	return entry.until.Sub(now)
}

func (b *backoffTracker) Schedule(id string, now time.Time, suggested time.Duration) time.Duration {
	entry := b.items[id]
	duration := suggested
	if duration <= 0 {
		if entry.duration == 0 {
			duration = b.min
		} else {
			duration = time.Duration(float64(entry.duration) * b.factor)
		}
	}
	if duration > b.max {
		duration = b.max
	}
	entry.duration = duration
	entry.until = now.Add(duration)
	b.items[id] = entry
	return duration
}

func (b *backoffTracker) Reset(id string) {
	delete(b.items, id)
}

func sqlNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Valid: true, Time: t}
}
