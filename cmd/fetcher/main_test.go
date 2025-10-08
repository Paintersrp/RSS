package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"

	"github.com/mmcdole/gofeed"

	"courier/internal/feed"
	"courier/internal/search"
	"courier/internal/store"
)

type stubFeedStore struct {
	updates       []store.UpdateFeedCrawlStateParams
	upserts       []store.UpsertItemParams
	feeds         []store.Feed
	upsertResults []store.UpsertItemResult
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
	result := store.UpsertItemResult{
		Item: store.Item{
			ID:          "item",
			FeedID:      arg.FeedID,
			FeedTitle:   "",
			Title:       arg.Title,
			ContentText: arg.ContentText,
			URL:         arg.URL,
		},
		Indexed: true,
	}
	if len(s.upsertResults) > 0 {
		result = s.upsertResults[0]
		s.upsertResults = s.upsertResults[1:]
		if result.Item.ID == "" {
			result.Item.ID = "item"
		}
		if result.Item.FeedID == "" {
			result.Item.FeedID = arg.FeedID
		}
		if result.Item.FeedTitle == "" {
			result.Item.FeedTitle = ""
		}
		if result.Item.Title == "" {
			result.Item.Title = arg.Title
		}
		if result.Item.ContentText == "" {
			result.Item.ContentText = arg.ContentText
		}
		if result.Item.URL == "" {
			result.Item.URL = arg.URL
		}
	}
	return result, nil
}

func (s *stubFeedStore) ListFeeds(ctx context.Context, active bool) ([]store.Feed, error) {
	feeds := make([]store.Feed, len(s.feeds))
	copy(feeds, s.feeds)
	return feeds, nil
}

type stubSearchClient struct {
	docCalls      [][]search.Document
	batchCalls    [][]search.Document
	batchErr      error
	batchErrLimit int
	batchErrCount int
	docErrs       []error
}

func (s *stubSearchClient) UpsertDocuments(ctx context.Context, docs []search.Document) error {
	copied := make([]search.Document, len(docs))
	copy(copied, docs)
	s.docCalls = append(s.docCalls, copied)
	if len(s.docErrs) > 0 {
		err := s.docErrs[0]
		s.docErrs = s.docErrs[1:]
		return err
	}
	return nil
}

func (s *stubSearchClient) UpsertBatch(ctx context.Context, docs []search.Document) error {
	copied := make([]search.Document, len(docs))
	copy(copied, docs)
	s.batchCalls = append(s.batchCalls, copied)
	if s.batchErr != nil {
		s.batchErrCount++
		if s.batchErrLimit == 0 || s.batchErrCount <= s.batchErrLimit {
			return s.batchErr
		}
	}
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

	result, docs := FetchFeed(ctx, repo, searchClient, fetcher, backoffs, feedRecord)
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
	if len(docs) != 0 {
		t.Fatalf("expected returned documents to match item upserts, got %d", len(docs))
	}
	if len(searchClient.docCalls) != 0 || len(searchClient.batchCalls) != 0 {
		t.Fatalf(
			"expected no search calls during fetch, got doc=%d batch=%d",
			len(searchClient.docCalls),
			len(searchClient.batchCalls),
		)
	}

	feedRecord.ETag = sql.NullString{Valid: true, String: etag}
	feedRecord.LastModified = sql.NullString{Valid: true, String: lastModified}

	second, moreDocs := FetchFeed(ctx, repo, searchClient, fetcher, backoffs, feedRecord)
	if second.Status != http.StatusNotModified {
		t.Fatalf("expected status 304, got %d", second.Status)
	}
	if second.Mutated {
		t.Fatalf("did not expect mutation on 304")
	}
	if len(moreDocs) != 0 {
		t.Fatalf("expected no documents returned for not modified feed, got %d", len(moreDocs))
	}
	if len(searchClient.docCalls) != 0 || len(searchClient.batchCalls) != 0 {
		t.Fatalf(
			"expected no search calls during fetches, got doc=%d batch=%d",
			len(searchClient.docCalls),
			len(searchClient.batchCalls),
		)
	}
	if len(repo.updates) != 1 {
		t.Fatalf("expected no additional update call, got %d", len(repo.updates))
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

	result, docs := FetchFeed(ctx, repo, searchClient, fetcher, newBackoffTracker(), feedRecord)
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
	if len(docs) != 1 {
		t.Fatalf("expected one document returned, got %d", len(docs))
	}
	if docs[0].URL != wantURL {
		t.Fatalf("document URL = %q, want %q", docs[0].URL, wantURL)
	}
	if len(searchClient.docCalls) != 0 || len(searchClient.batchCalls) != 0 {
		t.Fatalf(
			"expected no search calls during fetch, got doc=%d batch=%d",
			len(searchClient.docCalls),
			len(searchClient.batchCalls),
		)
	}
}

func TestFetchFeedSanitizesContentText(t *testing.T) {
	repo := &stubFeedStore{}
	searchClient := &stubSearchClient{}
	fetcher := &stubFetcher{responses: []fetchResponse{{
		result: feed.Result{
			Status: http.StatusOK,
			Feed: &gofeed.Feed{
				Items: []*gofeed.Item{{
					Title:   "Example post",
					Content: `<p>Hello&nbsp;<em>world</em>!<script>bad()</script></p>`,
				}},
			},
		},
	}}}

	ctx := context.Background()
	feedRecord := store.Feed{ID: "feed-1", URL: "http://example.com/feed"}

	result, docs := FetchFeed(ctx, repo, searchClient, fetcher, newBackoffTracker(), feedRecord)
	if result.Status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.Status)
	}
	if len(repo.upserts) != 1 {
		t.Fatalf("expected one item upsert, got %d", len(repo.upserts))
	}

	const wantContent = "Hello world!"
	if repo.upserts[0].ContentText != wantContent {
		t.Fatalf("stored content text = %q, want %q", repo.upserts[0].ContentText, wantContent)
	}

	if len(docs) != 1 {
		t.Fatalf("expected one document returned, got %d", len(docs))
	}
	if docs[0].ContentText != wantContent {
		t.Fatalf("document content text = %q, want %q", docs[0].ContentText, wantContent)
	}
	if len(searchClient.docCalls) != 0 || len(searchClient.batchCalls) != 0 {
		t.Fatalf(
			"expected no search calls during fetch, got doc=%d batch=%d",
			len(searchClient.docCalls),
			len(searchClient.batchCalls),
		)
	}
}

func TestRunIndexesOnlyChangedItems(t *testing.T) {
	cases := []struct {
		name        string
		indexed     bool
		wantBatches int
	}{
		{name: "unchanged", indexed: false, wantBatches: 0},
		{name: "mutated", indexed: true, wantBatches: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubFeedStore{
				feeds: []store.Feed{{
					ID:     "feed-1",
					URL:    "http://example.com/feed",
					Active: true,
				}},
				upsertResults: []store.UpsertItemResult{{
					Item: store.Item{
						ID:          "item-1",
						FeedID:      "feed-1",
						FeedTitle:   "Example Feed",
						Title:       "Example post",
						ContentText: "body",
						URL:         "http://example.com/post",
					},
					Indexed: tc.indexed,
				}},
			}
			searchClient := &stubSearchClient{}
			fetcher := &stubFetcher{responses: []fetchResponse{{
				result: feed.Result{
					Status: http.StatusOK,
					Feed: &gofeed.Feed{Items: []*gofeed.Item{{
						Title:   "Example post",
						Content: "body",
						Link:    "http://example.com/post",
					}}},
				},
			}}}

			ctx := context.Background()
			run(ctx, "fetcher", repo, searchClient, fetcher, newBackoffTracker(), 10)

			if len(repo.upserts) != 1 {
				t.Fatalf("expected one item upsert, got %d", len(repo.upserts))
			}
			if got := len(searchClient.batchCalls); got != tc.wantBatches {
				t.Fatalf("batch calls = %d, want %d", got, tc.wantBatches)
			}
			if tc.wantBatches == 0 {
				if len(searchClient.docCalls) != 0 {
					t.Fatalf("expected no fallback upserts, got %d", len(searchClient.docCalls))
				}
				return
			}
			if len(searchClient.batchCalls[0]) != 1 {
				t.Fatalf("expected one document in batch, got %d", len(searchClient.batchCalls[0]))
			}
			doc := searchClient.batchCalls[0][0]
			if doc.ID != "item-1" {
				t.Fatalf("document ID = %q, want %q", doc.ID, "item-1")
			}
			if doc.Title != "Example post" {
				t.Fatalf("document title = %q, want %q", doc.Title, "Example post")
			}
			if doc.URL != "http://example.com/post" {
				t.Fatalf("document URL = %q, want %q", doc.URL, "http://example.com/post")
			}
		})
	}
}

func TestRunFallsBackToSingleDocumentUpserts(t *testing.T) {
	makeFetcher := func() *stubFetcher {
		return &stubFetcher{responses: []fetchResponse{{
			result: feed.Result{
				Status: http.StatusOK,
				Feed: &gofeed.Feed{Items: []*gofeed.Item{
					{
						Title: "First",
						Link:  "https://example.com/first",
					},
					{
						Title: "Second",
						Link:  "https://example.com/second",
					},
				}},
			},
		}}}
	}

	t.Run("indexes documents individually", func(t *testing.T) {
		repo := &stubFeedStore{
			feeds: []store.Feed{{
				ID:     "feed-1",
				URL:    "http://example.com/feed",
				Active: true,
			}},
		}
		searchClient := &stubSearchClient{
			batchErr:      errors.New("batch failed"),
			batchErrLimit: 1,
		}

		ctx := context.Background()
		run(ctx, "fetcher", repo, searchClient, makeFetcher(), newBackoffTracker(), 10)

		if len(searchClient.batchCalls) != 1 {
			t.Fatalf("expected one batch attempt, got %d", len(searchClient.batchCalls))
		}
		if len(searchClient.docCalls) != 2 {
			t.Fatalf("expected two fallback upserts, got %d", len(searchClient.docCalls))
		}
		gotTitles := []string{searchClient.docCalls[0][0].Title, searchClient.docCalls[1][0].Title}
		wantTitles := []string{"First", "Second"}
		for i, want := range wantTitles {
			if gotTitles[i] != want {
				t.Fatalf("fallback doc %d title = %q, want %q", i, gotTitles[i], want)
			}
		}
	})

	t.Run("skips failing document but continues", func(t *testing.T) {
		repo := &stubFeedStore{
			feeds: []store.Feed{{
				ID:     "feed-1",
				URL:    "http://example.com/feed",
				Active: true,
			}},
		}
		searchClient := &stubSearchClient{
			batchErr:      errors.New("batch failed"),
			batchErrLimit: 2,
			docErrs:       []error{errors.New("doc failed"), nil},
		}

		ctx := context.Background()
		run(ctx, "fetcher", repo, searchClient, makeFetcher(), newBackoffTracker(), 10)

		if len(searchClient.batchCalls) != 2 {
			t.Fatalf("expected two batch attempts, got %d", len(searchClient.batchCalls))
		}
		if len(searchClient.docCalls) != 2 {
			t.Fatalf("expected two fallback upserts, got %d", len(searchClient.docCalls))
		}
		if got := searchClient.docCalls[0][0].Title; got != "First" {
			t.Fatalf("first fallback doc title = %q, want %q", got, "First")
		}
		if got := searchClient.docCalls[1][0].Title; got != "Second" {
			t.Fatalf("second fallback doc title = %q, want %q", got, "Second")
		}
	})
}

// Ensure stub satisfies interfaces at compile time.
var _ feedStore = (*stubFeedStore)(nil)
var _ feedRepository = (*stubFeedStore)(nil)
var _ documentIndexer = (*stubSearchClient)(nil)
var _ feedFetcher = (*stubFetcher)(nil)
