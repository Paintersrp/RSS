-- name: InsertFeed :one
INSERT INTO feeds (url)
VALUES ($1)
ON CONFLICT (url) DO NOTHING
RETURNING id, url, title, etag, last_modified, last_crawled, active;

-- name: ListFeeds :many
SELECT id, url, title, etag, last_modified, last_crawled, active
FROM feeds
WHERE active = $1
ORDER BY title ASC, url ASC;

-- name: UpdateFeedCrawlState :one
UPDATE feeds
SET etag = $2,
    last_modified = $3,
    last_crawled = $4,
    title = COALESCE(NULLIF($5, ''), title)
WHERE id = $1
RETURNING id, url, title, etag, last_modified, last_crawled, active;
