package item

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"courier/internal/item/htmlclean"
	"courier/internal/item/urlcanon"
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
	url := urlcanon.Normalize(firstNonEmpty(fi.Link))

	contentHTML := strings.TrimSpace(fi.Content)
	if contentHTML == "" {
		contentHTML = strings.TrimSpace(fi.Description)
	}

	contentText := htmlclean.CleanHTML(contentHTML, 2000)
	if contentText == "" {
		contentText = htmlclean.CleanHTML(fi.Description, 2000)
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
