package item

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	htmlstd "html"
	"net/url"
	"path"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"

	"github.com/mmcdole/gofeed"

	"courier/internal/store"
)

func HashContent(parts ...string) []byte {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return h.Sum(nil)
}

func ContentHashString(parts ...string) string {
	sum := HashContent(parts...)
	return hex.EncodeToString(sum)
}

func FromFeedItem(feedID string, fi *gofeed.Item) store.UpsertItemParams {
	guid := sql.NullString{}
	if fi.GUID != "" {
		guid.Valid = true
		guid.String = fi.GUID
	}

	author := sql.NullString{}
	if fi.Author != nil && fi.Author.Name != "" {
		author.Valid = true
		author.String = fi.Author.Name
	}

	published := sql.NullTime{}
	if fi.PublishedParsed != nil {
		published.Valid = true
		published.Time = fi.PublishedParsed.UTC()
	}

	retrieved := sql.NullTime{Valid: true, Time: time.Now().UTC()}

	title := strings.TrimSpace(fi.Title)
	url := normalizeURL(firstNonEmpty(fi.Link))

	contentHTML := strings.TrimSpace(fi.Content)
	if contentHTML == "" {
		contentHTML = strings.TrimSpace(fi.Description)
	}

	contentText := sanitizeHTML(contentHTML)
	if contentText == "" {
		contentText = strings.TrimSpace(fi.Description)
	}

	guidValue := ""
	if guid.Valid {
		guidValue = guid.String
	}

	hash := HashContent(feedID, guidValue, url, title, contentText)

	return store.UpsertItemParams{
		FeedID:      feedID,
		GUID:        guid,
		URL:         url,
		Title:       title,
		Author:      author,
		ContentHTML: contentHTML,
		ContentText: contentText,
		PublishedAt: published,
		RetrievedAt: retrieved,
		ContentHash: hash,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func sanitizeHTML(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	node, err := xhtml.Parse(strings.NewReader(trimmed))
	if err != nil {
		return collapseWhitespace(trimmed)
	}
	var builder strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode {
			switch strings.ToLower(n.Data) {
			case "script", "style":
				return
			}
		}
		if n.Type == xhtml.TextNode {
			text := collapseWhitespace(htmlstd.UnescapeString(n.Data))
			if text != "" {
				if builder.Len() > 0 {
					builder.WriteByte(' ')
				}
				builder.WriteString(text)
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return cleanupPunctuation(collapseWhitespace(htmlstd.UnescapeString(builder.String())))
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

var punctuationReplacer = strings.NewReplacer(
	" !", "!",
	" ?", "?",
	" ,", ",",
	" .", ".",
	" ;", ";",
	" :", ":",
)

func cleanupPunctuation(s string) string {
	return punctuationReplacer.Replace(s)
}

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return raw
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if port != "" {
		if (parsed.Scheme == "http" && port == "80") || (parsed.Scheme == "https" && port == "443") {
			port = ""
		}
	}
	if port != "" {
		parsed.Host = host + ":" + port
	} else {
		parsed.Host = host
	}
	if parsed.Path != "" {
		cleaned := path.Clean(parsed.Path)
		if cleaned == "." {
			cleaned = ""
		}
		parsed.Path = cleaned
	}
	parsed.Fragment = ""

	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" || lower == "mc_cid" || lower == "mc_eid" {
			query.Del(key)
		}
	}
	if len(query) == 0 {
		parsed.RawQuery = ""
	} else {
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}
