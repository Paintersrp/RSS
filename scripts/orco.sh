#!/usr/bin/env bash
set -euo pipefail
STACK="${1:-deploy/orco/stack.dev.yaml}"; shift || true
if command -v ./bin/orco >/dev/null 2>&1; then
  exec ./bin/orco --file "$STACK" "$@"
else
  exec orco --file "$STACK" "$@"
fi
