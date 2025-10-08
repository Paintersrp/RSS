package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"courier/internal/feed"
	"courier/internal/item"
	"courier/internal/item/urlcanon"
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
	logx.Info(svc, "crawl tick", nil)

	feeds, err := repo.ListFeeds(ctx, true)
	if err != nil {
		logx.Error(svc, "list feeds", err, nil)
		return
	}

	for _, f := range feeds {
		result := FetchFeed(ctx, repo, searchClient, fetcher, backoffs, f)

		extra := map[string]any{
			"feed":    f.URL,
			"feed_id": f.ID,
		}
		if result.Status != 0 {
			extra["status"] = result.Status
		}
		if result.Items > 0 {
			extra["items"] = result.Items
		}
		if result.Mutated {
			extra["mutated"] = true
		}
		if result.Reason != "" {
			extra["reason"] = result.Reason
		}
		if result.RetryIn > 0 {
			extra["retry_in"] = result.RetryIn.String()
		}

		switch {
		case result.Err != nil && errors.Is(result.Err, ErrBackoffActive):
			logx.Info(svc, "feed skipped", extra)
		case result.Err != nil:
			logx.Error(svc, "feed error", result.Err, extra)
		case result.Skipped:
			logx.Info(svc, "feed skipped", extra)
		default:
			logx.Info(svc, "feed processed", extra)
		}
	}
}

type feedStore interface {
	UpdateFeedCrawlState(context.Context, store.UpdateFeedCrawlStateParams) (store.Feed, error)
	UpsertItem(context.Context, store.UpsertItemParams) (store.UpsertItemResult, error)
}

type feedFetcher interface {
	Fetch(ctx context.Context, url, etag, lastModified string) (feed.Result, error)
}

type documentIndexer interface {
	UpsertDocuments(ctx context.Context, docs []search.Document) error
}

type FetchFeedResult struct {
	FeedID  string
	FeedURL string
	Status  int
	Items   int
	Mutated bool
	Err     error
	RetryIn time.Duration
	Skipped bool
	Reason  string
}

var ErrBackoffActive = errors.New("backoff active")

func FetchFeed(ctx context.Context, repo feedStore, searchClient documentIndexer, fetcher feedFetcher, backoffs *backoffTracker, f store.Feed) FetchFeedResult {
	result := FetchFeedResult{FeedID: f.ID, FeedURL: f.URL}

	now := time.Now().UTC()
	if wait := backoffs.Remaining(f.ID, now); wait > 0 {
		result.Err = ErrBackoffActive
		result.RetryIn = wait
		result.Skipped = true
		result.Reason = "backoff active"
		return result
	}

	etag := ""
	if f.ETag.Valid {
		etag = f.ETag.String
	}
	lastModified := ""
	if f.LastModified.Valid {
		lastModified = f.LastModified.String
	}

	res, err := fetcher.Fetch(ctx, f.URL, etag, lastModified)
	result.Status = res.Status
	if err != nil {
		result.Err = err
		if errors.Is(err, feed.ErrRetryLater) {
			duration := backoffs.Schedule(f.ID, now, res.RetryAfter)
			result.RetryIn = duration
			result.Reason = "retry scheduled"
		} else {
			result.Reason = "fetch failed"
		}
		return result
	}

	backoffs.Reset(f.ID)

	if res.Status == http.StatusNotModified {
		result.Skipped = true
		result.Reason = "not modified"
		return result
	}

	if res.Feed == nil {
		result.Skipped = true
		result.Reason = "no content"
		return result
	}

	updateTime := time.Now().UTC()
	title := f.Title
	if res.Feed != nil && res.Feed.Title != "" {
		title = res.Feed.Title
	}

	if _, err := repo.UpdateFeedCrawlState(ctx, store.UpdateFeedCrawlStateParams{
		ID:           f.ID,
		ETag:         sqlNullString(res.ETag),
		LastModified: sqlNullString(res.LastModified),
		LastCrawled:  sqlNullTime(updateTime),
		Title:        title,
	}); err != nil {
		result.Err = err
		result.Reason = "update feed"
		return result
	}

	result.Mutated = true

	var docs []search.Document
	for _, entry := range res.Feed.Items {
		params := item.FromFeedItem(f.ID, entry)
		params.URL = urlcanon.Normalize(params.URL)
		output, err := repo.UpsertItem(ctx, params)
		if err != nil {
			wrapped := fmt.Errorf("upsert item: %w", err)
			if result.Err != nil {
				result.Err = errors.Join(result.Err, wrapped)
			} else {
				result.Err = wrapped
			}
			if result.Reason == "" {
				result.Reason = "item upsert"
			}
			continue
		}
		doc := search.Document{
			ID:          output.Item.ID,
			FeedID:      output.Item.FeedID,
			FeedTitle:   output.Item.FeedTitle,
			Title:       output.Item.Title,
			ContentText: output.Item.ContentText,
			URL:         output.Item.URL,
		}
		if output.Item.PublishedAt.Valid {
			t := output.Item.PublishedAt.Time.UTC()
			doc.PublishedAt = &t
		}
		docs = append(docs, doc)
	}

	result.Items = len(docs)
	if err := searchClient.UpsertDocuments(ctx, docs); err != nil {
		result.Err = err
		result.Reason = "search upsert"
		return result
	}

	return result
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
