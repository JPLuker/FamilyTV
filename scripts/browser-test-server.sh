#!/usr/bin/env bash
set -eu
cd "$(dirname "$0")/.."
state=$(mktemp -d)
trap 'rm -rf "$state"' EXIT
(cd backend && go build -o "$state/server" ./cmd/server)
export FAMILYTV_STATE_PATH="$state/state.json" FAMILYTV_WEB_DIR="$PWD/web/dist" FAMILYTV_LISTEN=127.0.0.1:17881
"$state/server" &
pid=$!
trap 'kill "$pid" 2>/dev/null || true' TERM INT
wait "$pid"
