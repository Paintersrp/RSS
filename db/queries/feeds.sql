-- name: InsertFeed :one
INSERT INTO feeds (url)
VALUES (sqlc.arg(url))
ON CONFLICT (url) DO NOTHING
RETURNING id, url, title, etag, last_modified, last_crawled, active;

-- name: ListFeeds :many
SELECT id, url, title, etag, last_modified, last_crawled, active
FROM feeds
WHERE active = sqlc.arg(active)
ORDER BY title ASC, url ASC;

-- name: UpdateFeedCrawlState :one
UPDATE feeds
SET etag = sqlc.arg(etag),
    last_modified = sqlc.arg(last_modified),
    last_crawled = COALESCE(sqlc.arg(last_crawled), last_crawled),
    title = COALESCE(NULLIF(sqlc.arg(new_title)::text, ''), title)
WHERE id = sqlc.arg(id)
RETURNING id, url, title, etag, last_modified, last_crawled, active;
