package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetcherPreservesHeadersOnNotModified(t *testing.T) {
	var count int
	const (
		etag         = "W/\"123\""
		lastModified = "Mon, 02 Jan 2006 15:04:05 GMT"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.Header().Set("ETag", etag)
		w.Header().Set("Last-Modified", lastModified)
		if count == 1 {
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>Test</title></channel></rss>`))
			return
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	t.Cleanup(srv.Close)

	fetcher := NewFetcher()
	ctx := context.Background()

	res, err := fetcher.Fetch(ctx, srv.URL, "", "")
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if res.Status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Status)
	}
	if res.ETag != etag {
		t.Fatalf("expected etag %q, got %q", etag, res.ETag)
	}
	if res.LastModified != lastModified {
		t.Fatalf("expected last modified %q, got %q", lastModified, res.LastModified)
	}

	res, err = fetcher.Fetch(ctx, srv.URL, res.ETag, res.LastModified)
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if res.Status != http.StatusNotModified {
		t.Fatalf("expected status 304, got %d", res.Status)
	}
	if res.ETag != etag {
		t.Fatalf("expected etag %q on 304, got %q", etag, res.ETag)
	}
	if res.LastModified != lastModified {
		t.Fatalf("expected last modified %q on 304, got %q", lastModified, res.LastModified)
	}
}
