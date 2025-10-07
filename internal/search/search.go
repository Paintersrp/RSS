package search

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type Client struct {
	svc       string
	client    meilisearch.ServiceManager
	index     string
	batchSize int
}

// Option configures the search client.
type Option func(*Client)

const defaultBatchSize = 100

// WithBatchSize overrides the number of documents sent per Meilisearch
// upsert call. Values <= 0 fallback to the default chunk size.
func WithBatchSize(size int) Option {
	return func(c *Client) {
		c.batchSize = size
	}
}

func New(url string, opts ...Option) *Client {
	client := &Client{
		svc:       "search",
		client:    meilisearch.New(url),
		index:     "items",
		batchSize: defaultBatchSize,
	}
	for _, opt := range opts {
		opt(client)
	}
	if client.batchSize <= 0 {
		client.batchSize = defaultBatchSize
	}
	return client
}

func (c *Client) EnsureIndex(ctx context.Context) error {
	if _, err := c.client.GetIndexWithContext(ctx, c.index); err != nil {
		var apiErr *meilisearch.Error
		if errors.As(err, &apiErr) && apiErr.MeilisearchApiError.Code == "index_not_found" {
			if _, err := c.client.CreateIndexWithContext(ctx, &meilisearch.IndexConfig{Uid: c.index, PrimaryKey: "id"}); err != nil {
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
	if _, err := c.client.Index(c.index).UpdateSettingsWithContext(ctx, settings); err != nil {
		return err
	}
	return nil
}

func (c *Client) Health(ctx context.Context) error {
	if !c.client.IsHealthy() {
		return fmt.Errorf("meili unhealthy")
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

func (c *Client) Search(ctx context.Context, query string, limit, offset int, filters SearchFilters) (SearchResponse, error) {
	req := &meilisearch.SearchRequest{
		Offset: int64(offset),
		Limit:  int64(limit),
	}
	if filters.FeedID != "" {
		req.Filter = fmt.Sprintf("feed_id = \"%s\"", filters.FeedID)
	}

	res, err := c.client.Index(c.index).SearchWithContext(ctx, query, req)
	if err != nil {
		return SearchResponse{}, err
	}
	hits := make([]Document, 0, len(res.Hits))
	for _, hit := range res.Hits {
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
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				doc.PublishedAt = &t
			}
		}
		hits = append(hits, doc)
	}
	return SearchResponse{Query: query, Limit: limit, Offset: offset, EstimatedTotal: res.EstimatedTotalHits, Hits: hits}, nil
}

func (c *Client) UpsertDocuments(ctx context.Context, docs []Document) error {
	if len(docs) == 0 {
		return nil
	}

	batchSize := c.batchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	index := c.client.Index(c.index)
	for start := 0; start < len(docs); start += batchSize {
		end := start + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		if _, err := index.UpdateDocumentsWithContext(ctx, docs[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) IndexName() string {
	return c.index
}

func (c *Client) BatchSize() int {
	if c.batchSize <= 0 {
		return defaultBatchSize
	}
	return c.batchSize
}
