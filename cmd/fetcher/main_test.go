package main

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"github.com/mmcdole/gofeed"

	"courier/internal/feed"
	"courier/internal/search"
	"courier/internal/store"
)

type stubFeedStore struct {
	updates []store.UpdateFeedCrawlStateParams
	upserts []store.UpsertItemParams
}

func (s *stubFeedStore) UpdateFeedCrawlState(ctx context.Context, arg store.UpdateFeedCrawlStateParams) (store.Feed, error) {
	s.updates = append(s.updates, arg)
	return store.Feed{
		ID:           arg.ID,
		URL:          "http://example.com/feed",
		Title:        arg.Title,
		ETag:         arg.ETag,
		LastModified: arg.LastModified,
		LastCrawled:  arg.LastCrawled,
		Active:       true,
	}, nil
}

func (s *stubFeedStore) UpsertItem(ctx context.Context, arg store.UpsertItemParams) (store.UpsertItemResult, error) {
	s.upserts = append(s.upserts, arg)
	return store.UpsertItemResult{Item: store.Item{ID: "item", FeedID: arg.FeedID, FeedTitle: "", Title: arg.Title, ContentText: arg.ContentText, URL: arg.URL}}, nil
}

type stubSearchClient struct {
	calls [][]search.Document
}

func (s *stubSearchClient) UpsertDocuments(ctx context.Context, docs []search.Document) error {
	copied := make([]search.Document, len(docs))
	copy(copied, docs)
	s.calls = append(s.calls, copied)
	return nil
}

type fetchResponse struct {
	result feed.Result
	err    error
}

type stubFetcher struct {
	responses []fetchResponse
	calls     []fetchCall
}

type fetchCall struct {
	etag         string
	lastModified string
}

func (s *stubFetcher) Fetch(ctx context.Context, url, etag, lastModified string) (feed.Result, error) {
	s.calls = append(s.calls, fetchCall{etag: etag, lastModified: lastModified})
	if len(s.responses) == 0 {
		return feed.Result{}, nil
	}
	resp := s.responses[0]
	s.responses = s.responses[1:]
	return resp.result, resp.err
}

func TestFetchFeedSkipsUpdateOnNotModified(t *testing.T) {
	const (
		etag         = "W/\"123\""
		lastModified = "Mon, 02 Jan 2006 15:04:05 GMT"
	)

	repo := &stubFeedStore{}
	searchClient := &stubSearchClient{}
	fetcher := &stubFetcher{responses: []fetchResponse{
		{
			result: feed.Result{
				Status: http.StatusOK,
				Feed: &gofeed.Feed{
					Title: "Example",
					Items: []*gofeed.Item{},
				},
				ETag:         etag,
				LastModified: lastModified,
			},
		},
		{
			result: feed.Result{
				Status:       http.StatusNotModified,
				ETag:         etag,
				LastModified: lastModified,
			},
		},
	}}
	backoffs := newBackoffTracker()
	ctx := context.Background()
	feedRecord := store.Feed{ID: "feed-1", URL: "http://example.com/feed"}

	result := FetchFeed(ctx, repo, searchClient, fetcher, backoffs, feedRecord)
	if result.Status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.Status)
	}
	if !result.Mutated {
		t.Fatalf("expected feed to be mutated on first fetch")
	}
	if len(repo.updates) != 1 {
		t.Fatalf("expected one update call, got %d", len(repo.updates))
	}
	if repo.updates[0].ETag.String != etag {
		t.Fatalf("expected etag %q stored, got %q", etag, repo.updates[0].ETag.String)
	}
	if repo.updates[0].LastModified.String != lastModified {
		t.Fatalf("expected last modified %q stored, got %q", lastModified, repo.updates[0].LastModified.String)
	}
	if len(searchClient.calls) != 1 {
		t.Fatalf("expected one search upsert call, got %d", len(searchClient.calls))
	}

	feedRecord.ETag = sql.NullString{Valid: true, String: etag}
	feedRecord.LastModified = sql.NullString{Valid: true, String: lastModified}

	second := FetchFeed(ctx, repo, searchClient, fetcher, backoffs, feedRecord)
	if second.Status != http.StatusNotModified {
		t.Fatalf("expected status 304, got %d", second.Status)
	}
	if second.Mutated {
		t.Fatalf("did not expect mutation on 304")
	}
	if len(repo.updates) != 1 {
		t.Fatalf("expected no additional update call, got %d", len(repo.updates))
	}
	if len(searchClient.calls) != 1 {
		t.Fatalf("expected no additional search call, got %d", len(searchClient.calls))
	}
	if len(fetcher.calls) != 2 {
		t.Fatalf("expected two fetch calls, got %d", len(fetcher.calls))
	}
	if got := fetcher.calls[1]; got.etag != etag || got.lastModified != lastModified {
		t.Fatalf("expected conditional headers %q and %q, got %q and %q", etag, lastModified, got.etag, got.lastModified)
	}
}

func TestFetchFeedCanonicalizesItemURLs(t *testing.T) {
	repo := &stubFeedStore{}
	searchClient := &stubSearchClient{}
	fetcher := &stubFetcher{responses: []fetchResponse{{
		result: feed.Result{
			Status: http.StatusOK,
			Feed: &gofeed.Feed{
				Items: []*gofeed.Item{{
					Title: "Example post",
					Link:  " https://WWW.Example.com:443/posts/Go/?utm_source=rss&gclid=123#fragment ",
				}},
			},
		},
	}}}

	ctx := context.Background()
	feedRecord := store.Feed{ID: "feed-1", URL: "http://example.com/feed"}

	result := FetchFeed(ctx, repo, searchClient, fetcher, newBackoffTracker(), feedRecord)
	if result.Status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.Status)
	}
	if len(repo.upserts) != 1 {
		t.Fatalf("expected one item upsert, got %d", len(repo.upserts))
	}
	const wantURL = "https://example.com/posts/Go"
	if repo.upserts[0].URL != wantURL {
		t.Fatalf("stored URL = %q, want %q", repo.upserts[0].URL, wantURL)
	}
	if len(searchClient.calls) != 1 || len(searchClient.calls[0]) != 1 {
		t.Fatalf("expected one document upsert call with one document, got %+v", searchClient.calls)
	}
	if searchClient.calls[0][0].URL != wantURL {
		t.Fatalf("indexed URL = %q, want %q", searchClient.calls[0][0].URL, wantURL)
	}
}

// Ensure stub satisfies interfaces at compile time.
var _ feedStore = (*stubFeedStore)(nil)
var _ documentIndexer = (*stubSearchClient)(nil)
var _ feedFetcher = (*stubFetcher)(nil)
