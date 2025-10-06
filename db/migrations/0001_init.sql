-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL DEFAULT '',
    etag TEXT NULL,
    last_modified TEXT NULL,
    last_crawled TIMESTAMPTZ NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    guid TEXT NULL,
    url TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    author TEXT NULL,
    content_html TEXT NOT NULL DEFAULT '',
    content_text TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ NULL,
    retrieved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    content_hash BYTEA NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS feeds;
