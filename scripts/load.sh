#!/usr/bin/env bash
# Generate load against one scenario.
#
#   scripts/load.sh <cpu|leak|goroutine|lock|alloc> [fixed]
#
# Env overrides: APP_URL (default http://localhost:8080), DURATION (default 30s),
#                CONCURRENCY (default 20), REQUESTS (curl fallback, default 200)
#
# Uses `hey` when installed (go install github.com/rakyll/hey@latest),
# otherwise falls back to a parallel curl loop.
set -euo pipefail

SCENARIO="${1:?usage: load.sh <cpu|leak|goroutine|lock|alloc> [fixed]}"
MODE="${2:-}"
BASE="${APP_URL:-http://localhost:8080}"
DURATION="${DURATION:-30s}"
CONCURRENCY="${CONCURRENCY:-20}"
REQUESTS="${REQUESTS:-200}"

case "$SCENARIO" in
  cpu|leak|goroutine|lock|alloc) ;;
  *) echo "unknown scenario: $SCENARIO" >&2; exit 1 ;;
esac

URL="$BASE/api/$SCENARIO"
if [[ "$MODE" == "fixed" ]]; then
  URL="$URL?fixed=true"
fi

if [[ "$SCENARIO" == "leak" && "$MODE" != "fixed" ]]; then
  # Each /leak request retains 256 KiB. Never time-box this one with hey:
  # thousands of requests per second would fill memory in seconds.
  echo "leak (slow): sending exactly $REQUESTS requests (~$((REQUESTS / 4)) MiB retained)"
  seq 1 "$REQUESTS" | xargs -P 5 -I{} curl -s -o /dev/null "$URL"
  exit 0
fi

if command -v hey >/dev/null 2>&1; then
  echo "hey: $URL for $DURATION, $CONCURRENCY workers"
  hey -z "$DURATION" -c "$CONCURRENCY" "$URL"
else
  echo "hey not found; curl fallback: $REQUESTS requests, $CONCURRENCY parallel -> $URL"
  seq 1 "$REQUESTS" | xargs -P "$CONCURRENCY" -I{} curl -s -o /dev/null "$URL"
fi
