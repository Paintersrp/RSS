package httpx

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"courier/internal/logx"
	"courier/internal/search"
	"courier/internal/store"
)

type Config struct {
	Store   *store.Store
	Search  *search.Client
	DB      *sql.DB
	Service string
}

func NewServer(cfg Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(requestLogger(cfg.Service))

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
		offset := parseInt(c.QueryParam("offset"), 0)
		ctx := c.Request().Context()
		items, err := cfg.Store.ListRecent(ctx, store.ListRecentParams{Limit: int32(limit), Offset: int32(offset)})
		if err != nil {
			return err
		}
		views := make([]itemView, 0, len(items))
		for _, it := range items {
			views = append(views, mapItem(it))
		}
		return c.JSON(http.StatusOK, views)
	})

	e.GET("/search", func(c echo.Context) error {
		query := c.QueryParam("q")
		limit := parseInt(c.QueryParam("limit"), 20)
		offset := parseInt(c.QueryParam("offset"), 0)
		ctx := c.Request().Context()
		res, err := cfg.Search.Search(ctx, query, limit, offset)
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
