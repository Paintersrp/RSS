# Courier

Courier is an end-to-end RSS ingestion pipeline that stores normalized feed data in Postgres, indexes it in Meilisearch, and serves it via a Go API and React web client. The local stack is orchestrated with [orco](https://github.com/orco-run/orco) for single-box ergonomics.

## Prerequisites

- Go 1.22+
- Node.js 18+
- Docker (for Postgres and Meilisearch)
- orco binary available at `./bin/orco` or on your `$PATH`

## First run

```bash
just deps
just build
just up
```

The stack launches Postgres, Meilisearch, two API replicas, and the fetcher worker. Once ready you can:

- Open the web client at http://localhost:3000 (run `just web-dev` in a separate shell for live reload)
- Check API health at http://localhost:8080/healthz

Add a feed via curl:

```bash
curl -X POST http://localhost:8080/feeds \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://blog.rust-lang.org/feed.xml"}'
```

The fetcher checks feeds every `COURIER_EVERY` (2 minutes by default). Within a few minutes new items appear at `GET /items` and in the `/search` view.

### Useful commands

```
just status    # show service readiness
just logs      # follow aggregated service logs
just air       # run the API with hot reload
just web-dev   # start the Vite dev server
just goose-up  # apply migrations (requires COURIER_DSN)
```

### Troubleshooting

- **Ports in use** – ensure 5432, 7700, and 8080 are free or update the stack manifests.
- **Resetting data** – stop the stack and remove the Docker volumes created by Postgres/Meilisearch.
- **Meilisearch health** – check http://localhost:7700/health if `/healthz` reports search unavailable.

## Roadmap

- **v0.2**: tighter crawl backoff, improved content normalization, richer web UX (sorting, saved searches, skeletons).
- **v0.3**: observability upgrades (`/metrics`, richer ingest logs), resilience tweaks, dependency graph automation.
- **v0.4**: dual-index canary workflow with configurable Meili index selection and client surface to inspect served index.

Courier v0.1 focuses on delivering a complete baseline: migrations, typed data access, Meili index bootstrap, API surface, fetcher worker, and a React dashboard for browsing and searching items.
