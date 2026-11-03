#!/usr/bin/env bash
# Prove a memory leak with a heap diff:  snapshot -> load -> snapshot -> diff.
#
#   scripts/leak-diff.sh          # slow version: Add() shows up as growth
#   scripts/leak-diff.sh fixed    # fixed version: growth is ~0
set -euo pipefail

MODE="${1:-}"
PPROF="${PPROF_URL:-http://localhost:6060/debug/pprof}"
OUT="profiles/leak-diff-${MODE:-slow}"
mkdir -p "$OUT"

echo "1/3 baseline heap snapshot"
curl -sf -o "$OUT/before.prof" "$PPROF/heap?gc=1"

echo "2/3 sending load"
REQUESTS="${REQUESTS:-200}" bash "$(dirname "$0")/load.sh" leak $MODE >/dev/null

echo "3/3 second heap snapshot"
curl -sf -o "$OUT/after.prof" "$PPROF/heap?gc=1"

echo
echo "=== inuse_space growth between snapshots (positive = memory that stayed alive) ==="
go tool pprof -sample_index=inuse_space -top -nodecount=10 \
  -diff_base="$OUT/before.prof" "$OUT/after.prof"
