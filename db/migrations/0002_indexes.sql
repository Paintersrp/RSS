-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS items_identity_uniq
    ON items(feed_id, COALESCE(guid, url));

CREATE INDEX IF NOT EXISTS items_recent_idx ON items(published_at DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS items_feed_recent_idx ON items(feed_id, published_at DESC NULLS LAST);

-- +goose Down
DROP INDEX IF EXISTS items_feed_recent_idx;
DROP INDEX IF EXISTS items_recent_idx;
DROP INDEX IF EXISTS items_identity_uniq;
