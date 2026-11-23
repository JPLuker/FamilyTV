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
# Restore into a fresh database and compare the profile before touching the running stack.
"${compose[@]}" exec -T -u postgres db pg_dump -Fc -d familytv > "$tmp/backup.dump"
"${compose[@]}" exec -T -u postgres db createdb --owner=familytv recovery
"${compose[@]}" exec -T -u postgres db pg_restore --exit-on-error --single-transaction -d recovery < "$tmp/backup.dump"
profile=$("${compose[@]}" exec -T -u postgres db psql -At -d recovery -c "SELECT document->'profiles'->0->>'name' FROM familytv_household")
test "$profile" = "Smoke test"
"${compose[@]}" up -d --force-recreate
wait_ready
curl -fsS -b "$tmp/cookies" "$base/api/profiles" | jq -e '.profiles | length == 1 and .[0].name == "Smoke test" and .[0].captions'
curl -fsS -b "$tmp/cookies" -X POST "$base/api/logout" | jq -e '.authenticated == false'
test "$(curl -sS -b "$tmp/cookies" -o /dev/null -w '%{http_code}' "$base/api/admin")" = 401
