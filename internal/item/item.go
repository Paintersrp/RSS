package item

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

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
	url := strings.TrimSpace(firstNonEmpty(fi.Link))

	contentHTML := strings.TrimSpace(fi.Content)
	if contentHTML == "" {
		contentHTML = strings.TrimSpace(fi.Description)
	}

	contentText := strings.TrimSpace(stripHTML(contentHTML))

	hash := HashContent(feedID, guid.String, url, title, contentText)

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

func stripHTML(s string) string {
	replacer := strings.NewReplacer("<", " ", ">", " ")
	cleaned := replacer.Replace(s)
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	cleaned = strings.ReplaceAll(cleaned, "\r", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned
}
