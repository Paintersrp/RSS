package urlcanon

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "mixed case scheme and host with tracking",
			in:   " HTTPS://Example.COM:443/posts/Go/?utm_source=rss&utm_medium=feed&ref=keep#gclid ",
			want: "https://example.com/posts/Go?ref=keep",
		},
		{
			name: "strips leading www and root slash",
			in:   "https://www.Example.com/",
			want: "https://example.com",
		},
		{
			name: "removes default port",
			in:   "http://WWW.example.com:80/path",
			want: "http://example.com/path",
		},
		{
			name: "keeps non default port",
			in:   "https://www.example.com:8443/path/",
			want: "https://example.com:8443/path",
		},
		{
			name: "drops trailing slash before query",
			in:   "https://example.com/foo/?utm_source=feed&keep=1",
			want: "https://example.com/foo?keep=1",
		},
		{
			name: "removes trailing root slash with query",
			in:   "https://www.example.com/?utm_source=feed&Keep=1",
			want: "https://example.com?Keep=1",
		},
		{
			name: "removes tracking parameters with varied casing",
			in:   "https://example.com/path/?ID=42&Fbclid=abc&utm_campaign=test&GCLID=123",
			want: "https://example.com/path?ID=42",
		},
		{
			name: "collapses redundant path segments",
			in:   "https://example.com/a/b/../c/?utm_medium=feed",
			want: "https://example.com/a/c",
		},
		{
			name: "invalid url returned unchanged",
			in:   "not a url",
			want: "not a url",
		},
		{
			name: "empty input",
			in:   "   \t   ",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.in)
			if got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
