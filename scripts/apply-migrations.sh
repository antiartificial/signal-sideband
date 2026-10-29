#!/bin/sh
set -eu
test -n "$DB_PASSWORD"
for file in migrations/*.sql; do
  PGPASSWORD="$DB_PASSWORD" psql -w -U norn -d signal_sideband -v ON_ERROR_STOP=1 -f "$file"
done
