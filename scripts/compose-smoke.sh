#!/usr/bin/env bash
set -euo pipefail

# A separate project keeps the test away from household data.
tmp=$(mktemp -d)
project="familytv-smoke-${tmp##*.}"
project=${project,,}
compose=(docker compose --project-name "$project" -f compose.yaml -f "$tmp/ports.yaml")
cat > "$tmp/ports.yaml" <<'YAML'
services:
  api:
    ports: !override
      - "127.0.0.1::8080"
YAML
cleanup() {
  "${compose[@]}" down --volumes --remove-orphans
  rm -rf "$tmp"
}
trap cleanup EXIT
export FAMILYTV_MEDIA_ROOT="$tmp/media"
mkdir -p "$FAMILYTV_MEDIA_ROOT/shows/season"
wait_ready() {
  base="http://$("${compose[@]}" port api 8080)"
  curl --silent --show-error --fail --retry 30 --retry-connrefused --retry-delay 1 "$base/api/health" | jq -e '.status == "ok"'
}
"${compose[@]}" up -d --build
wait_ready
curl -fsS "$base/" > "$tmp/index"
grep -q 'id="root"' "$tmp/index"
curl -fsS "$base/api/setup/status" | jq -e '.needs_setup'
credentials='{"password":"temporary smoke test password"}'
curl -fsS -H 'Content-Type: application/json' -d "$credentials" "$base/api/setup" | jq -e '.created'
curl -fsS -c "$tmp/cookies" -H 'Content-Type: application/json' -d "$credentials" "$base/api/login" | jq -e '.authenticated'
curl -fsS -b "$tmp/cookies" -H 'Content-Type: application/json' -d '{"name":"Smoke test","captions":true}' "$base/api/profiles" | jq -e '.captions'
# Use the same PostgreSQL and read-only media mount as a household installation.
"${compose[@]}" exec -T api ffmpeg -v error -f lavfi -i color=c=blue:s=32x32:d=1 -c:v mpeg4 /tmp/clip.mp4
"${compose[@]}" exec -T api cat /tmp/clip.mp4 > "$FAMILYTV_MEDIA_ROOT/shows/season/clip.mp4"
cp "$FAMILYTV_MEDIA_ROOT/shows/season/clip.mp4" "$FAMILYTV_MEDIA_ROOT/duplicate.mp4"
curl -fsS -b "$tmp/cookies" -X POST "$base/api/media/library-scan" >/dev/null
for _ in {1..30}; do
  result=$(curl -fsS -b "$tmp/cookies" "$base/api/media/library-scan")
  if jq -e '.running == false' <<<"$result" >/dev/null; then break; fi
  sleep 1
done
jq -e '.running == false and .imported == 1 and .skipped == 1 and .failed == 0' <<<"$result"
curl -fsS -b "$tmp/cookies" "$base/api/media" | jq -e '.media | length == 1 and (.[0].review.profiles | length == 0)'
child=$(curl -fsS -b "$tmp/cookies" "$base/api/profiles" | jq -er '.profiles[0].id')
curl -fsS -b "$tmp/cookies" -X POST "$base/api/devices" |
  jq -ce '{code}' | curl -fsS -c "$tmp/device" -H 'Content-Type: application/json' --data-binary @- "$base/api/watch/join" | jq -e '.joined'
curl -fsS -b "$tmp/device" "$base/api/watch/device" | jq -e '.paired and (.active == false)'
device=$(curl -fsS -b "$tmp/cookies" "$base/api/devices" | jq -er '.devices[0].id')
jq -nc --arg device "$device" --arg profile "$child" '{device:$device,profile:$profile}' |
  curl -fsS -b "$tmp/cookies" -X PATCH -H 'Content-Type: application/json' --data-binary @- "$base/api/devices" | jq -e '.assigned'
# Restore into a fresh database and compare the profile before touching the running stack.
"${compose[@]}" exec -T -u postgres db pg_dump -Fc -d familytv > "$tmp/backup.dump"
"${compose[@]}" exec -T -u postgres db createdb --owner=familytv recovery
"${compose[@]}" exec -T -u postgres db pg_restore --exit-on-error --single-transaction -d recovery < "$tmp/backup.dump"
profile=$("${compose[@]}" exec -T -u postgres db psql -At -d recovery -c "SELECT document->'profiles'->0->>'name' FROM familytv_household")
test "$profile" = "Smoke test"
curl -fsS -b "$tmp/cookies" -X PUT -H 'Content-Type: application/json' -d '{"path":"shows"}' "$base/api/media/library-folder" | jq -e '.selected == "shows"'
# A new unique file outside the chosen folder must stay out after container recreation.
"${compose[@]}" exec -T api ffmpeg -v error -f lavfi -i color=c=green:s=32x32:d=1 -c:v mpeg4 /tmp/outside.mp4
"${compose[@]}" exec -T api cat /tmp/outside.mp4 > "$FAMILYTV_MEDIA_ROOT/outside.mp4"
"${compose[@]}" exec -T api ffmpeg -v error -f lavfi -i color=c=red:s=32x32:d=1 -c:v mpeg4 /tmp/another.mp4
"${compose[@]}" exec -T api cat /tmp/another.mp4 > "$FAMILYTV_MEDIA_ROOT/shows/season/new.mp4"
"${compose[@]}" up -d --force-recreate
wait_ready
for _ in {1..30}; do
  if curl -fsS -b "$tmp/cookies" "$base/api/media" | jq -e '.media | length == 2' >/dev/null; then break; fi
  sleep 1
done
curl -fsS -b "$tmp/cookies" "$base/api/media/library-folder" | jq -e '.selected == "shows"'
curl -fsS -b "$tmp/cookies" "$base/api/media" | jq -e '.media | any(.[]; .path == "shows/season/new.mp4") and length == 2 and all(.[]; (.review.profiles | length) == 0)'
curl -fsS -b "$tmp/cookies" "$base/api/profiles" | jq -e '.profiles | length == 1 and .[0].name == "Smoke test" and .[0].captions'
curl -fsS -b "$tmp/cookies" -H 'Content-Type: application/json' -d '{"mode":"manual"}' "$base/api/profiles/$child/session" >/dev/null
curl -fsS -b "$tmp/device" "$base/api/watch" | jq -e '.items | length == 0'
curl -fsS -b "$tmp/cookies" -X DELETE "$base/api/profiles/$child/session" >/dev/null
curl -fsS -b "$tmp/device" "$base/api/watch/device" | jq -e '.paired and (.active == false)'
curl -fsS -b "$tmp/cookies" -X POST "$base/api/logout" | jq -e '.authenticated == false'
test "$(curl -sS -b "$tmp/cookies" -o /dev/null -w '%{http_code}' "$base/api/admin")" = 401
