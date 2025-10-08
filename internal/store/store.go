package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
	metrics Metrics
}

type Metrics interface {
	ObserveDB(method string, err error, duration time.Duration)
}

func New(db *sql.DB, metrics Metrics) *Store {
	return &Store{
		db:      db,
		queries: sqlc.New(db),
		metrics: metrics,
	}
}

type ItemSortField string

const (
	ItemSortFieldPublishedAt ItemSortField = "published_at"
	ItemSortFieldRetrievedAt ItemSortField = "retrieved_at"
)

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)

type FilterItemsParams struct {
	FeedIDs       []string
	SortField     ItemSortField
	SortDirection SortDirection
	Limit         int32
	Offset        int32
}

type FilterItemsResult struct {
	Items []Item
	Total int64
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

func (s *Store) InsertFeed(ctx context.Context, url string) (feed Feed, err error) {
	if s.metrics != nil {
		defer func(start time.Time) {
			s.metrics.ObserveDB("InsertFeed", err, time.Since(start))
		}(time.Now())
	}

	row, err := s.queries.InsertFeed(ctx, url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrFeedExists
		}
		return Feed{}, err
	}
	feed = mapFeed(row)
	return feed, nil
}

func (s *Store) ListFeeds(ctx context.Context, active bool) (feeds []Feed, err error) {
	if s.metrics != nil {
		defer func(start time.Time) {
			s.metrics.ObserveDB("ListFeeds", err, time.Since(start))
		}(time.Now())
	}

	rows, err := s.queries.ListFeeds(ctx, active)
	if err != nil {
		return nil, err
	}
	feeds = make([]Feed, 0, len(rows))
	for _, f := range rows {
		feeds = append(feeds, mapFeed(f))
	}
	return feeds, nil
}

type UpdateFeedCrawlStateParams struct {
	ID           string
	ETag         sql.NullString
	LastModified sql.NullString
	LastCrawled  sql.NullTime
	Title        string
}

func (s *Store) UpdateFeedCrawlState(ctx context.Context, arg UpdateFeedCrawlStateParams) (feed Feed, err error) {
	if s.metrics != nil {
		defer func(start time.Time) {
			s.metrics.ObserveDB("UpdateFeedCrawlState", err, time.Since(start))
		}(time.Now())
	}

	var feedID uuid.UUID
	feedID, err = uuid.Parse(arg.ID)
	if err != nil {
		return Feed{}, err
	}

	var updated sqlc.Feed
	updated, err = s.queries.UpdateFeedCrawlState(ctx, sqlc.UpdateFeedCrawlStateParams{
		ID:           feedID,
		Etag:         arg.ETag,
		LastModified: arg.LastModified,
		LastCrawled:  arg.LastCrawled,
		NewTitle:     arg.Title,
	})
	if err != nil {
		return Feed{}, err
	}
	feed = mapFeed(updated)
	return feed, nil
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

func (s *Store) UpsertItem(ctx context.Context, arg UpsertItemParams) (result UpsertItemResult, err error) {
	if s.metrics != nil {
		defer func(start time.Time) {
			s.metrics.ObserveDB("UpsertItem", err, time.Since(start))
		}(time.Now())
	}

	var feedID uuid.UUID
	feedID, err = uuid.Parse(arg.FeedID)
	if err != nil {
		return UpsertItemResult{}, err
	}

	var row sqlc.UpsertItemRow
	row, err = s.queries.UpsertItem(ctx, sqlc.UpsertItemParams{
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

	result = UpsertItemResult{Item: item, Fresh: row.Inserted, Indexed: indexed}
	return result, nil
}

type ListRecentParams struct {
	Limit  int32
	Offset int32
}

func (s *Store) ListRecent(ctx context.Context, arg ListRecentParams) (items []Item, err error) {
	if s.metrics != nil {
		defer func(start time.Time) {
			s.metrics.ObserveDB("ListRecent", err, time.Since(start))
		}(time.Now())
	}

	var rows []sqlc.ListRecentRow
	rows, err = s.queries.ListRecent(ctx, sqlc.ListRecentParams{
		Limit:  arg.Limit,
		Offset: arg.Offset,
	})
	if err != nil {
		return nil, err
	}

	items = make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapItem(row.ID, row.FeedID, row.FeedTitle, row.Guid, row.Url, row.Title, row.Author, row.ContentHtml, row.ContentText, row.PublishedAt, row.RetrievedAt))
	}
	return items, nil
}

func (s *Store) ListRecentFiltered(ctx context.Context, feedIDs []string, direction SortDirection, limit, offset int32) (items []Item, err error) {
	if s.metrics != nil {
		defer func(start time.Time) {
			s.metrics.ObserveDB("ListRecentFiltered", err, time.Since(start))
		}(time.Now())
	}

	sortDirection := direction
	if sortDirection == "" {
		sortDirection = SortDirectionDesc
	}

	if sortDirection != SortDirectionAsc && sortDirection != SortDirectionDesc {
		err = fmt.Errorf("store: invalid sort direction %q", sortDirection)
		return nil, err
	}

	var parsedIDs []uuid.UUID
	if len(feedIDs) > 0 {
		parsedIDs = make([]uuid.UUID, len(feedIDs))
		for i, id := range feedIDs {
			parsedIDs[i], err = uuid.Parse(id)
			if err != nil {
				return nil, err
			}
		}
	}

	var rows []sqlc.ListRecentFilteredRow
	rows, err = s.queries.ListRecentFiltered(ctx, sqlc.ListRecentFilteredParams{
		FeedIds:       parsedIDs,
		SortDirection: string(sortDirection),
		ResultLimit:   limit,
		ResultOffset:  offset,
	})
	if err != nil {
		return nil, err
	}

	items = make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapItem(row.ID, row.FeedID, row.FeedTitle, row.Guid, row.Url, row.Title, row.Author, row.ContentHtml, row.ContentText, row.PublishedAt, row.RetrievedAt))
	}
	return items, nil
}

func (s *Store) ListByFeed(ctx context.Context, feedID string, limit, offset int32) (items []Item, err error) {
	if s.metrics != nil {
		defer func(start time.Time) {
			s.metrics.ObserveDB("ListByFeed", err, time.Since(start))
		}(time.Now())
	}

	var id uuid.UUID
	id, err = uuid.Parse(feedID)
	if err != nil {
		return nil, err
	}

	var rows []sqlc.ListByFeedRow
	rows, err = s.queries.ListByFeed(ctx, sqlc.ListByFeedParams{
		FeedID: id,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	items = make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapItem(row.ID, row.FeedID, row.FeedTitle, row.Guid, row.Url, row.Title, row.Author, row.ContentHtml, row.ContentText, row.PublishedAt, row.RetrievedAt))
	}
	return items, nil
}

func (s *Store) FilterItems(ctx context.Context, arg FilterItemsParams) (result FilterItemsResult, err error) {
	if s.metrics != nil {
		defer func(start time.Time) {
			s.metrics.ObserveDB("FilterItems", err, time.Since(start))
		}(time.Now())
	}

	sortField := arg.SortField
	if sortField == "" {
		sortField = ItemSortFieldPublishedAt
	}
	sortDirection := arg.SortDirection
	if sortDirection == "" {
		sortDirection = SortDirectionDesc
	}

	orderColumn := map[ItemSortField]string{
		ItemSortFieldPublishedAt: "i.published_at",
		ItemSortFieldRetrievedAt: "i.retrieved_at",
	}
	column, ok := orderColumn[sortField]
	if !ok {
		err = fmt.Errorf("store: invalid sort field %q", sortField)
		return FilterItemsResult{}, err
	}

	dirSQL := "DESC"
	switch sortDirection {
	case SortDirectionAsc:
		dirSQL = "ASC"
	case SortDirectionDesc:
		dirSQL = "DESC"
	default:
		err = fmt.Errorf("store: invalid sort direction %q", sortDirection)
		return FilterItemsResult{}, err
	}

	var builder strings.Builder
	builder.WriteString("SELECT i.id, i.feed_id, f.title AS feed_title, i.guid, i.url, i.title, i.author, i.content_html, i.content_text, i.published_at, i.retrieved_at, COUNT(*) OVER () AS total FROM items i JOIN feeds f ON f.id = i.feed_id")

	args := make([]any, 0, len(arg.FeedIDs)+2)
	placeholder := 1
	if len(arg.FeedIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(arg.FeedIDs))
		for _, id := range arg.FeedIDs {
			var parsed uuid.UUID
			parsed, err = uuid.Parse(id)
			if err != nil {
				return FilterItemsResult{}, err
			}
			ids = append(ids, parsed)
		}

		builder.WriteString(" WHERE i.feed_id IN (")
		for i, id := range ids {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("$%d", placeholder))
			placeholder++
			args = append(args, id)
		}
		builder.WriteString(")")
	}

	builder.WriteString(" ORDER BY ")
	if sortField == ItemSortFieldPublishedAt {
		builder.WriteString(fmt.Sprintf("%s %s NULLS LAST, i.retrieved_at DESC", column, dirSQL))
	} else {
		builder.WriteString(fmt.Sprintf("%s %s, i.id DESC", column, dirSQL))
	}

	builder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", placeholder, placeholder+1))
	args = append(args, arg.Limit, arg.Offset)

	rows, err := s.db.QueryContext(ctx, builder.String(), args...)
	if err != nil {
		return FilterItemsResult{}, err
	}
	defer rows.Close()

	items := make([]Item, 0)
	var total int64
	for rows.Next() {
		var (
			id          uuid.UUID
			feedID      uuid.UUID
			feedTitle   string
			guid        sql.NullString
			url         string
			title       string
			author      sql.NullString
			contentHTML string
			contentText string
			publishedAt sql.NullTime
			retrievedAt time.Time
			rowTotal    int64
		)
		if err = rows.Scan(&id, &feedID, &feedTitle, &guid, &url, &title, &author, &contentHTML, &contentText, &publishedAt, &retrievedAt, &rowTotal); err != nil {
			return FilterItemsResult{}, err
		}
		total = rowTotal
		items = append(items, mapItem(id, feedID, feedTitle, guid, url, title, author, contentHTML, contentText, publishedAt, retrievedAt))
	}
	if err = rows.Err(); err != nil {
		return FilterItemsResult{}, err
	}

	result = FilterItemsResult{Items: items, Total: total}
	return result, nil
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
