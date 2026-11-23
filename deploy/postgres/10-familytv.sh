#!/bin/sh
set -eu
psql -v ON_ERROR_STOP=1 --username postgres --dbname familytv <<'SQL'
CREATE ROLE familytv LOGIN;
GRANT CONNECT ON DATABASE familytv TO familytv;
GRANT USAGE, CREATE ON SCHEMA public TO familytv;
SQL
cat > "$PGDATA/pg_hba.conf" <<'HBA'
local all postgres peer
local familytv familytv trust
local all all reject
HBA
