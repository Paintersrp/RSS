package item

import (
        "testing"

        "github.com/mmcdole/gofeed"
)

func TestFromFeedItemNormalizesURL(t *testing.T) {
	feedItem := &gofeed.Item{Link: " https://WWW.Example.com:443/posts/Go/?utm_source=rss&fbclid=abc#section "}

	params := FromFeedItem("feed-1", feedItem)

	const want = "https://example.com/posts/Go"
	if params.URL != want {
		t.Fatalf("FromFeedItem URL = %q, want %q", params.URL, want)
	}
}

func TestFromFeedItemSanitizesHTML(t *testing.T) {
        feedItem := &gofeed.Item{Content: `<div><p>Hello <strong>world</strong>!<script>bad()</script></p><p>Second&nbsp;line</p></div>`}

        params := FromFeedItem("feed-1", feedItem)

        const want = "Hello world! Second line"
        if params.ContentText != want {
                t.Fatalf("FromFeedItem ContentText = %q, want %q", params.ContentText, want)
        }
}
