package feed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mmcdole/gofeed"
)

type Result struct {
	Status       int
	Feed         *gofeed.Feed
	ETag         string
	LastModified string
	RetryAfter   time.Duration
}

type Fetcher struct {
	client *http.Client
	parser *gofeed.Parser
}

func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: 20 * time.Second},
		parser: gofeed.NewParser(),
	}
}

func (f *Fetcher) Fetch(ctx context.Context, url, etag, lastModified string) (Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, err
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	res := Result{Status: resp.StatusCode}
	if resp.StatusCode == http.StatusNotModified {
		res.ETag = etag
		res.LastModified = lastModified
		return res, nil
	}

	if resp.StatusCode != http.StatusOK {
		if IsRetryable(resp.StatusCode) {
			res.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
			return res, ErrRetryLater
		}

		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return res, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	feed, err := f.parser.Parse(resp.Body)
	if err != nil {
		return res, err
	}

	res.Feed = feed
	res.ETag = resp.Header.Get("ETag")
	res.LastModified = resp.Header.Get("Last-Modified")
	if res.ETag == "" {
		res.ETag = etag
	}
	if res.LastModified == "" {
		res.LastModified = lastModified
	}
	return res, nil
}

var ErrRetryLater = errors.New("retry later")

func IsRetryable(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		diff := time.Until(t)
		if diff > 0 {
			return diff
		}
	}
	return 0
}
