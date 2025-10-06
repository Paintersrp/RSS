package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrFeedExists = errors.New("feed already exists")
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

type Feed struct {
	ID           string         `json:"id"`
	URL          string         `json:"url"`
	Title        string         `json:"title"`
	ETag         sql.NullString `json:"etag"`
	LastModified sql.NullString `json:"last_modified"`
	LastCrawled  sql.NullTime   `json:"last_crawled"`
	Active       bool           `json:"active"`
}

func (s *Store) InsertFeed(ctx context.Context, url string) (Feed, error) {
	const q = `INSERT INTO feeds (url) VALUES ($1)
ON CONFLICT (url) DO NOTHING
RETURNING id, url, title, etag, last_modified, last_crawled, active`
	var f Feed
	err := s.db.QueryRowContext(ctx, q, url).Scan(
		&f.ID, &f.URL, &f.Title, &f.ETag, &f.LastModified, &f.LastCrawled, &f.Active,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Feed{}, ErrFeedExists
		}
		return Feed{}, err
	}
	return f, nil
}

func (s *Store) ListFeeds(ctx context.Context, active bool) ([]Feed, error) {
	const q = `SELECT id, url, title, etag, last_modified, last_crawled, active
FROM feeds
WHERE active = $1
ORDER BY title ASC, url ASC`
	rows, err := s.db.QueryContext(ctx, q, active)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []Feed
	for rows.Next() {
		var f Feed
		if err := rows.Scan(&f.ID, &f.URL, &f.Title, &f.ETag, &f.LastModified, &f.LastCrawled, &f.Active); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}

type UpdateFeedCrawlStateParams struct {
	ID           string
	ETag         sql.NullString
	LastModified sql.NullString
	LastCrawled  sql.NullTime
	Title        string
}

func (s *Store) UpdateFeedCrawlState(ctx context.Context, arg UpdateFeedCrawlStateParams) (Feed, error) {
	const q = `UPDATE feeds
SET etag = $2,
    last_modified = $3,
    last_crawled = $4,
    title = COALESCE(NULLIF($5, ''), title)
WHERE id = $1
RETURNING id, url, title, etag, last_modified, last_crawled, active`
	var f Feed
	err := s.db.QueryRowContext(ctx, q, arg.ID, arg.ETag, arg.LastModified, arg.LastCrawled, arg.Title).Scan(
		&f.ID, &f.URL, &f.Title, &f.ETag, &f.LastModified, &f.LastCrawled, &f.Active,
	)
	if err != nil {
		return Feed{}, err
	}
	return f, nil
}

type Item struct {
	ID          string         `json:"id"`
	FeedID      string         `json:"feed_id"`
	FeedTitle   string         `json:"feed_title"`
	GUID        sql.NullString `json:"guid"`
	URL         string         `json:"url"`
	Title       string         `json:"title"`
	Author      sql.NullString `json:"author"`
	ContentHTML string         `json:"content_html"`
	ContentText string         `json:"content_text"`
	PublishedAt sql.NullTime   `json:"published_at"`
	RetrievedAt time.Time      `json:"retrieved_at"`
}

type UpsertItemParams struct {
	FeedID      string
	GUID        sql.NullString
	URL         string
	Title       string
	Author      sql.NullString
	ContentHTML string
	ContentText string
	PublishedAt sql.NullTime
	RetrievedAt sql.NullTime
	ContentHash []byte
}

type UpsertItemResult struct {
	Item  Item
	Fresh bool
}

func (s *Store) UpsertItem(ctx context.Context, arg UpsertItemParams) (UpsertItemResult, error) {
	const q = `WITH upsert AS (
INSERT INTO items (
    feed_id, guid, url, title, author, content_html, content_text, published_at, retrieved_at, content_hash
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, now()), $10
) ON CONFLICT (feed_id, COALESCE(guid, url)) DO UPDATE SET
    url = EXCLUDED.url,
    title = EXCLUDED.title,
    author = EXCLUDED.author,
    content_html = EXCLUDED.content_html,
    content_text = EXCLUDED.content_text,
    published_at = EXCLUDED.published_at,
    retrieved_at = EXCLUDED.retrieved_at,
    content_hash = EXCLUDED.content_hash
RETURNING id, feed_id, guid, url, title, author, content_html, content_text, published_at, retrieved_at, content_hash, xmax = 0 AS inserted
)
SELECT u.id, u.feed_id, f.title AS feed_title, u.guid, u.url, u.title, u.author, u.content_html, u.content_text, u.published_at, u.retrieved_at, u.inserted
FROM upsert u
JOIN feeds f ON f.id = u.feed_id`

	row := s.db.QueryRowContext(ctx, q,
		arg.FeedID,
		arg.GUID,
		arg.URL,
		arg.Title,
		arg.Author,
		arg.ContentHTML,
		arg.ContentText,
		arg.PublishedAt,
		arg.RetrievedAt,
		arg.ContentHash,
	)

	var res UpsertItemResult
	var inserted bool
	if err := row.Scan(
		&res.Item.ID,
		&res.Item.FeedID,
		&res.Item.FeedTitle,
		&res.Item.GUID,
		&res.Item.URL,
		&res.Item.Title,
		&res.Item.Author,
		&res.Item.ContentHTML,
		&res.Item.ContentText,
		&res.Item.PublishedAt,
		&res.Item.RetrievedAt,
		&inserted,
	); err != nil {
		return UpsertItemResult{}, err
	}
	res.Fresh = inserted
	return res, nil
}

type ListRecentParams struct {
	Limit  int32
	Offset int32
}

func (s *Store) ListRecent(ctx context.Context, arg ListRecentParams) ([]Item, error) {
	const q = `SELECT i.id, i.feed_id, f.title AS feed_title, i.guid, i.url, i.title, i.author, i.content_html, i.content_text, i.published_at, i.retrieved_at
FROM items i
JOIN feeds f ON f.id = i.feed_id
ORDER BY i.published_at DESC NULLS LAST, i.retrieved_at DESC
LIMIT $1 OFFSET $2`
	rows, err := s.db.QueryContext(ctx, q, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.FeedID, &item.FeedTitle, &item.GUID, &item.URL, &item.Title, &item.Author, &item.ContentHTML, &item.ContentText, &item.PublishedAt, &item.RetrievedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListByFeed(ctx context.Context, feedID string, limit, offset int32) ([]Item, error) {
	const q = `SELECT i.id, i.feed_id, f.title AS feed_title, i.guid, i.url, i.title, i.author, i.content_html, i.content_text, i.published_at, i.retrieved_at
FROM items i
JOIN feeds f ON f.id = i.feed_id
WHERE i.feed_id = $1
ORDER BY i.published_at DESC NULLS LAST, i.retrieved_at DESC
LIMIT $2 OFFSET $3`
	rows, err := s.db.QueryContext(ctx, q, feedID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.FeedID, &item.FeedTitle, &item.GUID, &item.URL, &item.Title, &item.Author, &item.ContentHTML, &item.ContentText, &item.PublishedAt, &item.RetrievedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
