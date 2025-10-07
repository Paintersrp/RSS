package item

import "testing"

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "strips tracking params",
			in:   " https://Example.com:443/posts?id=1&utm_source=newsletter#fragment ",
			out:  "https://example.com/posts?id=1",
		},
		{
			name: "normalizes default ports",
			in:   "http://example.com:80/path/",
			out:  "http://example.com/path",
		},
		{
			name: "preserves meaningful query",
			in:   "https://example.com/search?q=Go+lang&ref=home",
			out:  "https://example.com/search?q=Go+lang&ref=home",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeURL(tc.in); got != tc.out {
				t.Fatalf("normalizeURL(%q) = %q, want %q", tc.in, got, tc.out)
			}
		})
	}
}

func TestSanitizeHTML(t *testing.T) {
	input := `<p>Hello <strong>world</strong></p><script>alert('xss')</script>`
	if got := sanitizeHTML(input); got != "Hello world" {
		t.Fatalf("unexpected sanitizeHTML result: %q", got)
	}
}
