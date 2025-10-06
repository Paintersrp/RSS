#!/usr/bin/env bash
set -euo pipefail
HOST="${1:?host:port required}"
TIMEOUT="${2:-60}"
start=$(date +%s)
while true; do
  if nc -z ${HOST%:*} ${HOST#*:} >/dev/null 2>&1; then
    exit 0
  fi
  now=$(date +%s)
  if (( now - start > TIMEOUT )); then
    echo "timeout waiting for $HOST" >&2
    exit 1
  fi
  sleep 1
done
