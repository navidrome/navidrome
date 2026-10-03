#!/usr/bin/env sh
# Runs the Music Journey end-to-end smoke test against a throwaway instance of
# the dev Navidrome-compatible harness. Starts the harness on a free port,
# waits for it, runs the checks, then tears it down. Exit code = test result.
set -e

PORT="${MOCK_PORT:-4699}"
DIR="$(cd "$(dirname "$0")/.." && pwd)"

MOCK_PORT="$PORT" node "$DIR/tools/dev-navidrome-mock.mjs" &
MOCK_PID=$!
trap 'kill $MOCK_PID 2>/dev/null || true' EXIT INT TERM

# Wait for the harness to accept connections (max ~10s).
for _ in $(seq 1 50); do
  if node -e "fetch('http://localhost:$PORT/rest/ping').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))" 2>/dev/null; then
    break
  fi
  sleep 0.2
done

MOCK_URL="http://localhost:$PORT" node "$DIR/tools/journey-smoke.mjs"
