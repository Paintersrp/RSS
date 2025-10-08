package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"courier/internal/store"
)

type stubStore struct {
	filterItemsFunc func(context.Context, store.FilterItemsParams) ([]store.Item, error)
}

func (s *stubStore) ListFeeds(context.Context, bool) ([]store.Feed, error) {
	return nil, nil
}

func (s *stubStore) InsertFeed(context.Context, string) (store.Feed, error) {
	return store.Feed{}, nil
}

func (s *stubStore) FilterItems(ctx context.Context, params store.FilterItemsParams) ([]store.Item, error) {
	if s.filterItemsFunc != nil {
		return s.filterItemsFunc(ctx, params)
	}
	return nil, nil
}

func TestItemsHandlerValidPagination(t *testing.T) {
	t.Parallel()

	called := false
	stub := &stubStore{
		filterItemsFunc: func(ctx context.Context, params store.FilterItemsParams) ([]store.Item, error) {
			called = true
			if params.Limit != maxItemsLimit {
				t.Fatalf("expected limit %d, got %d", maxItemsLimit, params.Limit)
			}
			if params.Offset != 5 {
				t.Fatalf("expected offset 5, got %d", params.Offset)
			}
			return []store.Item{}, nil
		},
	}

	srv := NewServer(Config{Store: stub, Service: "test"})

	req := httptest.NewRequest(http.MethodGet, "/items?limit=1000&offset=5", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !called {
		t.Fatalf("expected FilterItems to be called")
	}
}

func TestItemsHandlerNegativeLimit(t *testing.T) {
	t.Parallel()

	stub := &stubStore{
		filterItemsFunc: func(ctx context.Context, params store.FilterItemsParams) ([]store.Item, error) {
			t.Fatalf("FilterItems should not be called for invalid limit")
			return nil, nil
		},
	}

	srv := NewServer(Config{Store: stub, Service: "test"})

	req := httptest.NewRequest(http.MethodGet, "/items?limit=-1", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestItemsHandlerNegativeOffset(t *testing.T) {
	t.Parallel()

	stub := &stubStore{
		filterItemsFunc: func(ctx context.Context, params store.FilterItemsParams) ([]store.Item, error) {
			t.Fatalf("FilterItems should not be called for invalid offset")
			return nil, nil
		},
	}

	srv := NewServer(Config{Store: stub, Service: "test"})

	req := httptest.NewRequest(http.MethodGet, "/items?offset=-10", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
