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

func TestSanitizeHTML(t *testing.T) {
	html := `<div><p>Hello <strong>world</strong>!<script>bad()</script></p><p>Second line</p></div>`
	got := sanitizeHTML(html)
	const want = "Hello world! Second line"
	if got != want {
		t.Fatalf("sanitizeHTML() = %q, want %q", got, want)
	}
}
