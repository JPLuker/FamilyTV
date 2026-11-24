#!/usr/bin/env bash
set -eu
cd "$(dirname "$0")/.."
state=$(mktemp -d)
trap 'rm -rf "$state"' EXIT
(cd backend && go build -o "$state/server" ./cmd/server)
mkdir "$state/media"
ffmpeg -v error -f lavfi -i color=c=blue:s=64x64:d=1 -c:v libvpx -threads 1 "$state/media/forest.webm"
printf "WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nForest\n" > "$state/media/forest.vtt"
# Small original fixtures exercise season and playlist ordering through the real scanner.
for entry in 'Sesame Street S01E03 C' 'Sesame Street S01E01 A' 'Sesame Street S00E01 Special' 'Lesson A' 'Lesson Z'; do
  ffmpeg -v error -i "$state/media/forest.webm" -f lavfi -i sine=duration=1 -map 0:v -map 1:a -c:v copy -c:a libopus -metadata title="$entry" "$state/media/$entry.webm"
done
printf '%s' '{"title":"Lesson A","channel":"Crash Course","playlist_title":"Science","playlist_index":2}' > "$state/media/Lesson A.info.json"
printf '%s' '{"title":"Lesson Z","channel":"Crash Course","playlist_title":"Science","playlist_index":1}' > "$state/media/Lesson Z.info.json"
# Only the test server sees this unavailable upstream. Go build has already completed.
export HTTPS_PROXY=http://127.0.0.1:1 NO_PROXY=127.0.0.1,localhost
export FAMILYTV_MEDIA_DIR="$state/media"
export FAMILYTV_STATE_PATH="$state/state.json" FAMILYTV_WEB_DIR="$PWD/web/dist" FAMILYTV_LISTEN=127.0.0.1:17881
"$state/server" &
pid=$!
trap 'kill "$pid" 2>/dev/null || true' TERM INT
wait "$pid"
