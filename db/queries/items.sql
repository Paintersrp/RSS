-- name: UpsertItem :one
WITH existing AS (
    SELECT i.id, i.content_hash
    FROM items i
    WHERE i.feed_id = sqlc.arg(feed_id)
      AND (
          i.guid = sqlc.arg(guid)
          OR (sqlc.arg(guid) IS NULL AND i.guid IS NULL AND i.url = sqlc.arg(url))
      )
),
upsert AS (
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
              content_hash,
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
       u.inserted,
       (u.inserted OR e.content_hash IS DISTINCT FROM u.content_hash) AS indexed
FROM upsert u
LEFT JOIN existing e ON e.id = u.id
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
