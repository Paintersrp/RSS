package httpx

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/google/uuid"

	"courier/internal/logx"
	"courier/internal/search"
	"courier/internal/store"
)

type storeAPI interface {
	ListFeeds(context.Context, bool) ([]store.Feed, error)
	InsertFeed(context.Context, string) (store.Feed, error)
	FilterItems(context.Context, store.FilterItemsParams) (store.FilterItemsResult, error)
}

type Config struct {
	Store   storeAPI
	Search  *search.Client
	DB      *sql.DB
	Service string
	Metrics *Metrics
}

const maxItemsLimit = 200

func NewServer(cfg Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(requestLogger(cfg.Service))
	if cfg.Metrics != nil {
		e.Use(cfg.Metrics.Middleware())
		handler := promhttp.HandlerFor(cfg.Metrics.Gatherer(), promhttp.HandlerOpts{})
		e.GET("/metrics", echo.WrapHandler(handler))
	}

	e.GET("/healthz", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()

		if err := cfg.DB.PingContext(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "db down"})
		}

		if err := cfg.Search.Health(ctx); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "search down"})
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.GET("/feeds", func(c echo.Context) error {
		ctx := c.Request().Context()
		feeds, err := cfg.Store.ListFeeds(ctx, true)
		if err != nil {
			return err
		}
		views := make([]feedView, 0, len(feeds))
		for _, f := range feeds {
			views = append(views, mapFeed(f))
		}
		return c.JSON(http.StatusOK, views)
	})

	type createFeedReq struct {
		URL string `json:"url"`
	}

	e.POST("/feeds", func(c echo.Context) error {
		var req createFeedReq
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid payload")
		}
		if req.URL == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "url required")
		}
		ctx := c.Request().Context()
		feed, err := cfg.Store.InsertFeed(ctx, req.URL)
		if err != nil {
			if errors.Is(err, store.ErrFeedExists) {
				return echo.NewHTTPError(http.StatusConflict, "feed exists")
			}
			return err
		}
		return c.JSON(http.StatusCreated, mapFeed(feed))
	})

	e.GET("/items", func(c echo.Context) error {
		limit := parseInt(c.QueryParam("limit"), 50)
		if limit < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "limit must be non-negative")
		}
		if limit > maxItemsLimit {
			limit = maxItemsLimit
		}

		offset := parseInt(c.QueryParam("offset"), 0)
		if offset < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "offset must be non-negative")
		}
		feedIDs := c.QueryParams()["feed_id"]
		for _, id := range feedIDs {
			if _, err := uuid.Parse(id); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid feed_id")
			}
		}
		sortParam := c.QueryParam("sort")
		if sortParam == "" {
			sortParam = "published_at:desc"
		}

		parts := strings.SplitN(sortParam, ":", 2)
		if len(parts) != 2 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sort parameter")
		}

		field := strings.ToLower(strings.TrimSpace(parts[0]))
		direction := strings.ToLower(strings.TrimSpace(parts[1]))

		var sortField store.ItemSortField
		switch field {
		case string(store.ItemSortFieldPublishedAt):
			sortField = store.ItemSortFieldPublishedAt
		case string(store.ItemSortFieldRetrievedAt):
			sortField = store.ItemSortFieldRetrievedAt
		default:
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sort field")
		}

		var sortDirection store.SortDirection
		switch direction {
		case string(store.SortDirectionAsc):
			sortDirection = store.SortDirectionAsc
		case string(store.SortDirectionDesc):
			sortDirection = store.SortDirectionDesc
		default:
			return echo.NewHTTPError(http.StatusBadRequest, "invalid sort direction")
		}

		ctx := c.Request().Context()
		result, err := cfg.Store.FilterItems(ctx, store.FilterItemsParams{
			FeedIDs:       feedIDs,
			SortField:     sortField,
			SortDirection: sortDirection,
			Limit:         int32(limit),
			Offset:        int32(offset),
		})
		if err != nil {
			return err
		}
		views := make([]itemView, 0, len(result.Items))
		for _, it := range result.Items {
			views = append(views, mapItem(it))
		}
		c.Response().Header().Set("X-Total-Count", strconv.FormatInt(result.Total, 10))
		return c.JSON(http.StatusOK, map[string]any{
			"items": views,
			"total": result.Total,
		})
	})

	e.GET("/search", func(c echo.Context) error {
		query := c.QueryParam("q")
		limit := parseInt(c.QueryParam("limit"), 20)
		offset := parseInt(c.QueryParam("offset"), 0)
		feedID := c.QueryParam("feed_id")
		ctx := c.Request().Context()
		res, err := cfg.Search.Search(ctx, query, limit, offset, search.SearchFilters{FeedID: feedID})
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, res)
	})

	return e
}

func parseInt(v string, def int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func requestLogger(service string) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:      true,
		LogMethod:       true,
		LogURI:          true,
		LogStatus:       true,
		LogError:        true,
		LogResponseSize: true,
		Skipper:         func(c echo.Context) bool { return false },
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			extra := map[string]any{
				"method":  v.Method,
				"uri":     v.URI,
				"status":  v.Status,
				"latency": v.Latency.String(),
				"size":    v.ResponseSize,
			}
			if v.Error != nil {
				logx.Error(service, "request", v.Error, extra)
			} else {
				logx.Info(service, "request", extra)
			}
			return nil
		},
	})
}

type feedView struct {
	ID           string     `json:"id"`
	URL          string     `json:"url"`
	Title        string     `json:"title"`
	ETag         *string    `json:"etag,omitempty"`
	LastModified *string    `json:"last_modified,omitempty"`
	LastCrawled  *time.Time `json:"last_crawled,omitempty"`
}

func mapFeed(f store.Feed) feedView {
	view := feedView{
		ID:    f.ID,
		URL:   f.URL,
		Title: f.Title,
	}
	if f.ETag.Valid {
		view.ETag = &f.ETag.String
	}
	if f.LastModified.Valid {
		view.LastModified = &f.LastModified.String
	}
	if f.LastCrawled.Valid {
		t := f.LastCrawled.Time.UTC()
		view.LastCrawled = &t
	}
	return view
}

type itemView struct {
	ID          string     `json:"id"`
	FeedID      string     `json:"feed_id"`
	FeedTitle   string     `json:"feed_title"`
	GUID        *string    `json:"guid,omitempty"`
	URL         string     `json:"url"`
	Title       string     `json:"title"`
	Author      *string    `json:"author,omitempty"`
	ContentHTML string     `json:"content_html"`
	ContentText string     `json:"content_text"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	RetrievedAt time.Time  `json:"retrieved_at"`
}

func mapItem(it store.Item) itemView {
	view := itemView{
		ID:          it.ID,
		FeedID:      it.FeedID,
		FeedTitle:   it.FeedTitle,
		URL:         it.URL,
		Title:       it.Title,
		ContentHTML: it.ContentHTML,
		ContentText: it.ContentText,
		RetrievedAt: it.RetrievedAt,
	}
	if it.GUID.Valid {
		view.GUID = &it.GUID.String
	}
	if it.Author.Valid {
		view.Author = &it.Author.String
	}
	if it.PublishedAt.Valid {
		t := it.PublishedAt.Time.UTC()
		view.PublishedAt = &t
	}
	return view
}
