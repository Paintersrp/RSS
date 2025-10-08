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
			in:   " HTTPS://Example.COM:443/posts/Go/?utm_source=rss&utm_medium=feed&ref=keep#fragment ",
			want: "https://example.com/posts/Go?ref=keep",
		},
		{
			name: "strips leading www and preserves root path",
			in:   "https://www.Example.com/",
			want: "https://example.com/",
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
			name: "drops trailing slash on non root path",
			in:   "https://example.com/foo/bar/",
			want: "https://example.com/foo/bar",
		},
		{
			name: "removes known tracking parameters",
			in:   "https://example.com/?id=42&fbclid=abc&utm_campaign=test&gclid=123",
			want: "https://example.com/?id=42",
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
