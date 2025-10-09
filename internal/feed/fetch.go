package feed

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"syscall"
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
		if isTransientFetchError(err) {
			return Result{}, fmt.Errorf("%w: %w", ErrTransientFetch, err)
		}
		return Result{}, err
	}
	defer resp.Body.Close()

	res := Result{Status: resp.StatusCode}
	currentETag := resp.Header.Get("ETag")
	currentLastModified := resp.Header.Get("Last-Modified")

	if resp.StatusCode == http.StatusNotModified {
		if currentETag != "" {
			res.ETag = currentETag
		} else {
			res.ETag = etag
		}
		if currentLastModified != "" {
			res.LastModified = currentLastModified
		} else {
			res.LastModified = lastModified
		}
		return res, nil
	}

	if resp.StatusCode != http.StatusOK {
		if isTransientStatus(resp.StatusCode) {
			res.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
			return res, fmt.Errorf("%w: http status %d", ErrTransientFetch, resp.StatusCode)
		}

		if isRateLimited(resp.StatusCode) {
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
	res.ETag = currentETag
	res.LastModified = currentLastModified
	if res.ETag == "" {
		res.ETag = etag
	}
	if res.LastModified == "" {
		res.LastModified = lastModified
	}
	return res, nil
}

var (
	ErrRetryLater     = errors.New("retry later")
	ErrTransientFetch = errors.New("transient fetch")
)

var transientSyscallErrors = []error{
	syscall.ECONNRESET,
	syscall.ECONNREFUSED,
	syscall.ECONNABORTED,
	syscall.EPIPE,
	syscall.EHOSTUNREACH,
	syscall.ENETUNREACH,
	syscall.ENETDOWN,
	syscall.ENETRESET,
	syscall.ETIMEDOUT,
}

func isRateLimited(status int) bool {
	return status == http.StatusTooManyRequests
}

func isTransientStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
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

func isTransientFetchError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	for _, target := range transientSyscallErrors {
		if errors.Is(err, target) {
			return true
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
		if netErr.Temporary() {
			return true
		}
	}

	if errors.Is(err, net.ErrClosed) {
		return true
	}

	return false
}
