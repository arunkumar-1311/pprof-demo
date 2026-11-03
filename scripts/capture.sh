#!/usr/bin/env bash
# Capture every profile type into one timestamped folder.
#
#   scripts/capture.sh [output-dir]
#
# Env overrides: PPROF_URL (default http://localhost:6060/debug/pprof),
#                DURATION  (CPU seconds, default 15)
#
# Start load in another terminal first, otherwise the CPU profile will be empty.
set -euo pipefail

PPROF="${PPROF_URL:-http://localhost:6060/debug/pprof}"
DURATION="${DURATION:-15}"
OUT="${1:-profiles/$(date +%Y%m%d-%H%M%S)}"
mkdir -p "$OUT"

fetch() { # fetch <name> <path>
  echo "  $1"
  curl -sf -o "$OUT/$1" "$PPROF/$2"
}

echo "Saving to $OUT"
fetch heap.prof        "heap?gc=1"          # live memory (GC first)
fetch allocs.prof      "allocs"             # all allocations since start
fetch goroutine.prof   "goroutine"
fetch goroutine.txt    "goroutine?debug=1"  # human-readable, grouped by stack
fetch goroutine-full.txt "goroutine?debug=2" # every goroutine with wait time
fetch mutex.prof       "mutex"
fetch block.prof       "block"
fetch threadcreate.prof "threadcreate"

echo "  cpu.prof (waiting ${DURATION}s, keep load running)"
curl -sf -o "$OUT/cpu.prof" "$PPROF/profile?seconds=$DURATION"

echo "  trace.out (3s)"
curl -sf -o "$OUT/trace.out" "$PPROF/trace?seconds=3"

echo
echo "Done. Open with:"
echo "  go tool pprof -http=:8081 $OUT/cpu.prof"
echo "  go tool pprof -top $OUT/heap.prof"
echo "  go tool trace $OUT/trace.out"
