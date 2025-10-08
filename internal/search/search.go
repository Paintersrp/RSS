package search

import (
	"context"
	"errors"
	"fmt"
	"time"

	"courier/internal/logx"
	meilisearch "github.com/meilisearch/meilisearch-go"
)

type Document struct {
	ID          string     `json:"id"`
	FeedID      string     `json:"feed_id"`
	FeedTitle   string     `json:"feed_title"`
	Title       string     `json:"title"`
	ContentText string     `json:"content_text"`
	URL         string     `json:"url"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type Metrics interface {
	ObserveSearch(method string, err error, duration time.Duration)
}

type Client struct {
	svc     string
	client  meilisearch.ServiceManager
	index   string
	metrics Metrics
}

func New(url string, metrics Metrics) *Client {
	return &Client{
		svc:     "search",
		client:  meilisearch.New(url),
		index:   "items",
		metrics: metrics,
	}
}

func (c *Client) EnsureIndex(ctx context.Context) (err error) {
	if c.metrics != nil {
		defer func(start time.Time) {
			c.metrics.ObserveSearch("EnsureIndex", err, time.Since(start))
		}(time.Now())
	}

	if _, err = c.client.GetIndexWithContext(ctx, c.index); err != nil {
		var apiErr *meilisearch.Error
		if errors.As(err, &apiErr) && apiErr.MeilisearchApiError.Code == "index_not_found" {
			if _, err = c.client.CreateIndexWithContext(ctx, &meilisearch.IndexConfig{Uid: c.index, PrimaryKey: "id"}); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	settings := &meilisearch.Settings{
		SearchableAttributes: []string{"title", "content_text"},
		FilterableAttributes: []string{"feed_id", "published_at"},
	}
	if _, err = c.client.Index(c.index).UpdateSettingsWithContext(ctx, settings); err != nil {
		return err
	}
	return nil
}

func (c *Client) Health(ctx context.Context) (err error) {
	if c.metrics != nil {
		defer func(start time.Time) {
			c.metrics.ObserveSearch("Health", err, time.Since(start))
		}(time.Now())
	}

	if !c.client.IsHealthy() {
		err = fmt.Errorf("meili unhealthy")
		return err
	}
	return nil
}

type SearchResponse struct {
	Query          string     `json:"query"`
	Limit          int        `json:"limit"`
	Offset         int        `json:"offset"`
	EstimatedTotal int64      `json:"estimated_total"`
	Hits           []Document `json:"hits"`
}

type SearchFilters struct {
	FeedID string
}

func (c *Client) Search(ctx context.Context, query string, limit, offset int, filters SearchFilters) (resp SearchResponse, err error) {
	if c.metrics != nil {
		defer func(start time.Time) {
			c.metrics.ObserveSearch("Search", err, time.Since(start))
		}(time.Now())
	}

	req := &meilisearch.SearchRequest{
		Offset: int64(offset),
		Limit:  int64(limit),
	}
	if filters.FeedID != "" {
		req.Filter = fmt.Sprintf("feed_id = \"%s\"", filters.FeedID)
	}

	var searchRes *meilisearch.SearchResponse
	searchRes, err = c.client.Index(c.index).SearchWithContext(ctx, query, req)
	if err != nil {
		return SearchResponse{}, err
	}
	hits := make([]Document, 0, len(searchRes.Hits))
	for _, hit := range searchRes.Hits {
		m, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}
		doc := Document{}
		if v, ok := m["id"].(string); ok {
			doc.ID = v
		}
		if v, ok := m["feed_id"].(string); ok {
			doc.FeedID = v
		}
		if v, ok := m["feed_title"].(string); ok {
			doc.FeedTitle = v
		}
		if v, ok := m["title"].(string); ok {
			doc.Title = v
		}
		if v, ok := m["content_text"].(string); ok {
			doc.ContentText = v
		}
		if v, ok := m["url"].(string); ok {
			doc.URL = v
		}
		if v, ok := m["published_at"].(string); ok && v != "" {
			if parsed, parseErr := time.Parse(time.RFC3339, v); parseErr == nil {
				doc.PublishedAt = &parsed
			}
		}
		hits = append(hits, doc)
	}
	resp = SearchResponse{Query: query, Limit: limit, Offset: offset, EstimatedTotal: searchRes.EstimatedTotalHits, Hits: hits}
	return resp, nil
}

func (c *Client) UpsertDocuments(ctx context.Context, docs []Document) (err error) {
	if c.metrics != nil {
		defer func(start time.Time) {
			c.metrics.ObserveSearch("UpsertDocuments", err, time.Since(start))
		}(time.Now())
	}

	if len(docs) == 0 {
		return nil
	}
	_, err = c.client.Index(c.index).UpdateDocumentsWithContext(ctx, docs)
	return err
}

func (c *Client) UpsertBatch(ctx context.Context, docs []Document) (err error) {
	if c.metrics != nil {
		defer func(start time.Time) {
			c.metrics.ObserveSearch("UpsertBatch", err, time.Since(start))
		}(time.Now())
	}

	if len(docs) == 0 {
		return nil
	}

	logx.Info(c.svc, "upsert batch", map[string]any{"index": c.index, "batch_size": len(docs)})

	_, err = c.client.Index(c.index).UpdateDocumentsWithContext(ctx, docs)
	return err
}

func (c *Client) IndexName() string {
	return c.index
}
