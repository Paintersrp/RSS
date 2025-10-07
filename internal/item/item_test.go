package item

import "testing"

func TestNormalizeURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "strips utm and fragment",
			in:   "https://Example.com:443/path?utm_source=test&ref=keep#section",
			want: "https://example.com/path?ref=keep",
		},
		{
			name: "removes default port and trims spaces",
			in:   "  http://BLOG.example.com:80/post?id=42&fbclid=abc  ",
			want: "http://blog.example.com/post?id=42",
		},
		{
			name: "handles invalid url",
			in:   "not a url",
			want: "not a url",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeURL(tc.in)
			if got != tc.want {
				t.Fatalf("normalizeURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
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
