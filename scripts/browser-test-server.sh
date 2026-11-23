#!/usr/bin/env bash
set -eu
cd "$(dirname "$0")/.."
state=$(mktemp -d)
trap 'rm -rf "$state"' EXIT
(cd backend && go build -o "$state/server" ./cmd/server)
mkdir "$state/media"
ffmpeg -v error -f lavfi -i color=c=blue:s=64x64:d=1 -c:v libvpx -threads 1 "$state/media/forest.webm"
printf "WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nForest\n" > "$state/media/forest.vtt"
export FAMILYTV_MEDIA_DIR="$state/media"
export FAMILYTV_STATE_PATH="$state/state.json" FAMILYTV_WEB_DIR="$PWD/web/dist" FAMILYTV_LISTEN=127.0.0.1:17881
"$state/server" &
pid=$!
trap 'kill "$pid" 2>/dev/null || true' TERM INT
wait "$pid"
