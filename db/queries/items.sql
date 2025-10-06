-- name: UpsertItem :one
WITH upsert AS (
    INSERT INTO items (
        feed_id,
        guid,
        url,
        title,
        author,
        content_html,
        content_text,
        published_at,
        retrieved_at,
        content_hash
    ) VALUES (
        sqlc.arg(feed_id),
        sqlc.arg(guid),
        sqlc.arg(url),
        sqlc.arg(title),
        sqlc.arg(author),
        sqlc.arg(content_html),
        sqlc.arg(content_text),
        sqlc.arg(published_at),
        COALESCE(sqlc.narg(retrieved_at), now()),
        sqlc.arg(content_hash)
    )
    ON CONFLICT (feed_id, COALESCE(guid, url)) DO UPDATE SET
        url = EXCLUDED.url,
        title = EXCLUDED.title,
        author = EXCLUDED.author,
        content_html = EXCLUDED.content_html,
        content_text = EXCLUDED.content_text,
        published_at = EXCLUDED.published_at,
        retrieved_at = EXCLUDED.retrieved_at,
        content_hash = EXCLUDED.content_hash
    RETURNING id,
              feed_id,
              guid,
              url,
              title,
              author,
              content_html,
              content_text,
              published_at,
              retrieved_at,
              xmax = 0 AS inserted
)
SELECT u.id,
       u.feed_id,
       f.title AS feed_title,
       u.guid,
       u.url,
       u.title,
       u.author,
       u.content_html,
       u.content_text,
       u.published_at,
       u.retrieved_at,
       u.inserted
FROM upsert u
JOIN feeds f ON f.id = u.feed_id;

-- name: ListRecent :many
SELECT i.id,
       i.feed_id,
       f.title AS feed_title,
       i.guid,
       i.url,
       i.title,
       i.author,
       i.content_html,
       i.content_text,
       i.published_at,
       i.retrieved_at
FROM items i
JOIN feeds f ON f.id = i.feed_id
ORDER BY i.published_at DESC NULLS LAST, i.retrieved_at DESC
LIMIT $1 OFFSET $2;

-- name: ListByFeed :many
SELECT i.id,
       i.feed_id,
       f.title AS feed_title,
       i.guid,
       i.url,
       i.title,
       i.author,
       i.content_html,
       i.content_text,
       i.published_at,
       i.retrieved_at
FROM items i
JOIN feeds f ON f.id = i.feed_id
WHERE i.feed_id = $1
ORDER BY i.published_at DESC NULLS LAST, i.retrieved_at DESC
LIMIT $2 OFFSET $3;
