# Use bash with strict flags
set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

# Variables
BIN := "./bin"
STACK ?= "deploy/orco/stack.dev.yaml"

# Default recipe
default: help

help:
    @echo "Common tasks:"
    @echo "  just deps       # install dev tools (air, goose, sqlc)"
    @echo "  just sqlc       # generate typed repos"
    @echo "  just goose-up   # apply DB migrations"
    @echo "  just build      # build api + fetcher"
    @echo "  just up         # run stack via orco"
    @echo "  just status     # orco status"
    @echo "  just logs       # tail all logs via orco"
    @echo "  just air        # dev run API with air"
    @echo "  just web-dev    # run Vite dev server"

deps:
    go install github.com/cosmtrek/air@latest
    go install github.com/pressly/goose/v3/cmd/goose@latest
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
    cd web && npm install

sqlc:
    sqlc generate

goose-up:
    : "${COURIER_DSN:?Set COURIER_DSN env (matches deploy/orco/secrets/api.env)}"
    goose -dir ./db/migrations postgres "$COURIER_DSN" up

build-api:
    mkdir -p {{BIN}}
    go build -o {{BIN}}/api ./cmd/api

build-fetcher:
    mkdir -p {{BIN}}
    go build -o {{BIN}}/fetcher ./cmd/fetcher

build: build-api build-fetcher

graph: build
    ./scripts/orco.sh "{{STACK}}" graph --dot > deploy/orco/graphs/latest.dot

status: build
    ./scripts/orco.sh "{{STACK}}" status

up: build
    ./scripts/orco.sh "{{STACK}}" up

logs:
    ./scripts/orco.sh "{{STACK}}" logs -f

restart svc:
    ./scripts/orco.sh "{{STACK}}" restart {{svc}}

air:
    air -c .air.toml

web-dev:
    cd web && npm run dev

# Convenience for first run
dev-all:
    just build
    just up
    just web-dev
