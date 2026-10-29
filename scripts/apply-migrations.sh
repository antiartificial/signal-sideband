#!/bin/sh
set -eu
test -n "$DB_PASSWORD"

if [ "$(PGPASSWORD="$DB_PASSWORD" psql -w -U norn -d signal_sideband -tAc "SELECT to_regclass('public.messages') IS NULL")" = "t" ]; then
  PGPASSWORD="$DB_PASSWORD" psql -w -U norn -d signal_sideband -v ON_ERROR_STOP=1 -f migrations/schema.sql
fi

for file in migrations/[0-9]*.sql; do
  PGPASSWORD="$DB_PASSWORD" psql -w -U norn -d signal_sideband -v ON_ERROR_STOP=1 -f "$file"
done
