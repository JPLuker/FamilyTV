# Household database backups

Run from the directory containing compose.yaml. The database contains parent credential hashes, active session credentials and child profile settings. Keep backups private. The legacy JSON volume remains read-only after import.

## Make and verify a backup

Use a new private directory for each backup. PostgreSQL takes a consistent snapshot while FamilyTV is running.

```sh
set -eu
umask 077
backup_dir=$(mktemp -d ./familytv-backup-XXXXXXXX)
docker compose exec -T -u postgres db pg_dump -Fc -d familytv > "$backup_dir/household.dump"
docker compose exec -T -u postgres db pg_restore --list < "$backup_dir/household.dump" > "$backup_dir/contents.txt"
```

Verify recovery by restoring the archive into a fresh database. The following command stops if the verification database already exists. The final query prints the schema version and number of profiles.

```sh
set -eu
docker compose exec -T -u postgres db createdb --owner=familytv familytv_verify
docker compose exec -T -u postgres db pg_restore --exit-on-error --single-transaction -d familytv_verify < "$backup_dir/household.dump"
docker compose exec -T -u postgres db psql -d familytv_verify -c "SELECT version, coalesce(jsonb_array_length(document->'profiles'),0) AS profiles FROM familytv_household"
```

## Recover on a fresh installation

Start only the database in a fresh Compose project with empty volumes. Restore before starting the API.

```sh
set -eu
docker compose up -d --wait db
docker compose exec -T -u postgres db pg_restore --exit-on-error --single-transaction -d familytv < "$backup_dir/household.dump"
docker compose up -d api
```

Sign in and check the profiles. Keep the original installation and verified archive until recovery is confirmed.
