package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"courier/internal/store/sqlc"
)

var (
	ErrFeedExists = errors.New("feed already exists")
)

type Store struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func New(db *sql.DB) *Store {
	return &Store{
		db:      db,
		queries: sqlc.New(db),
	}
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
	feed, err := s.queries.InsertFeed(ctx, url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Feed{}, ErrFeedExists
		}
		return Feed{}, err
	}
	return mapFeed(feed), nil
}

func (s *Store) ListFeeds(ctx context.Context, active bool) ([]Feed, error) {
	feeds, err := s.queries.ListFeeds(ctx, active)
	if err != nil {
		return nil, err
	}
	result := make([]Feed, 0, len(feeds))
	for _, f := range feeds {
		result = append(result, mapFeed(f))
	}
	return result, nil
}

type UpdateFeedCrawlStateParams struct {
	ID           string
	ETag         sql.NullString
	LastModified sql.NullString
	LastCrawled  sql.NullTime
	Title        string
}

func (s *Store) UpdateFeedCrawlState(ctx context.Context, arg UpdateFeedCrawlStateParams) (Feed, error) {
	feedID, err := uuid.Parse(arg.ID)
	if err != nil {
		return Feed{}, err
	}

	updated, err := s.queries.UpdateFeedCrawlState(ctx, sqlc.UpdateFeedCrawlStateParams{
		ID:           feedID,
		Etag:         arg.ETag,
		LastModified: arg.LastModified,
		LastCrawled:  arg.LastCrawled,
		NewTitle:     arg.Title,
	})
	if err != nil {
		return Feed{}, err
	}
	return mapFeed(updated), nil
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
	Item    Item
	Fresh   bool
	Indexed bool
}

func (s *Store) UpsertItem(ctx context.Context, arg UpsertItemParams) (UpsertItemResult, error) {
	feedID, err := uuid.Parse(arg.FeedID)
	if err != nil {
		return UpsertItemResult{}, err
	}

	row, err := s.queries.UpsertItem(ctx, sqlc.UpsertItemParams{
		FeedID:      feedID,
		Guid:        arg.GUID,
		Url:         arg.URL,
		Title:       arg.Title,
		Author:      arg.Author,
		ContentHtml: arg.ContentHTML,
		ContentText: arg.ContentText,
		PublishedAt: arg.PublishedAt,
		RetrievedAt: nullTimeArg(arg.RetrievedAt),
		ContentHash: arg.ContentHash,
	})
	if err != nil {
		return UpsertItemResult{}, err
	}

	item := mapItem(row.ID, row.FeedID, row.FeedTitle, row.Guid, row.Url, row.Title, row.Author, row.ContentHtml, row.ContentText, row.PublishedAt, row.RetrievedAt)

	indexed := row.Indexed.Valid && row.Indexed.Bool

	return UpsertItemResult{Item: item, Fresh: row.Inserted, Indexed: indexed}, nil
}

type ListRecentParams struct {
	Limit  int32
	Offset int32
}

func (s *Store) ListRecent(ctx context.Context, arg ListRecentParams) ([]Item, error) {
	rows, err := s.queries.ListRecent(ctx, sqlc.ListRecentParams{
		Limit:  arg.Limit,
		Offset: arg.Offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapItem(row.ID, row.FeedID, row.FeedTitle, row.Guid, row.Url, row.Title, row.Author, row.ContentHtml, row.ContentText, row.PublishedAt, row.RetrievedAt))
	}
	return items, nil
}

func (s *Store) ListByFeed(ctx context.Context, feedID string, limit, offset int32) ([]Item, error) {
	id, err := uuid.Parse(feedID)
	if err != nil {
		return nil, err
	}

	rows, err := s.queries.ListByFeed(ctx, sqlc.ListByFeedParams{
		FeedID: id,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapItem(row.ID, row.FeedID, row.FeedTitle, row.Guid, row.Url, row.Title, row.Author, row.ContentHtml, row.ContentText, row.PublishedAt, row.RetrievedAt))
	}
	return items, nil
}

func mapFeed(f sqlc.Feed) Feed {
	return Feed{
		ID:           f.ID.String(),
		URL:          f.Url,
		Title:        f.Title,
		ETag:         f.Etag,
		LastModified: f.LastModified,
		LastCrawled:  f.LastCrawled,
		Active:       f.Active,
	}
}

func mapItem(id uuid.UUID, feedID uuid.UUID, feedTitle string, guid sql.NullString, url string, title string, author sql.NullString, contentHTML string, contentText string, publishedAt sql.NullTime, retrievedAt time.Time) Item {
	return Item{
		ID:          id.String(),
		FeedID:      feedID.String(),
		FeedTitle:   feedTitle,
		GUID:        guid,
		URL:         url,
		Title:       title,
		Author:      author,
		ContentHTML: contentHTML,
		ContentText: contentText,
		PublishedAt: publishedAt,
		RetrievedAt: retrievedAt,
	}
}

func nullTimeArg(t sql.NullTime) interface{} {
	if t.Valid {
		return t.Time
	}
	return nil
}
