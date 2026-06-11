#!/usr/bin/env bash
# Apply SQL migrations to the configured Postgres database (idempotent).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"

run_sql_file() {
  local migration="$1"
  if command -v psql >/dev/null 2>&1; then
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$migration"
  else
    docker run --rm -i postgres:16-alpine psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f - <"$migration"
  fi
}

schema_check() {
  if command -v psql >/dev/null 2>&1; then
    psql "$DATABASE_URL" -tAc "SELECT to_regclass('public.manga_series')"
  else
    docker run --rm postgres:16-alpine psql "$DATABASE_URL" -tAc "SELECT to_regclass('public.manga_series')"
  fi
}

if schema_check | grep -q manga_series; then
  echo "Database schema already present; skipping migrations."
  exit 0
fi

echo "Applying database migrations..."
for migration in "$ROOT"/db/migrations/*.sql; do
  [ -f "$migration" ] || continue
  echo "  -> $(basename "$migration")"
  run_sql_file "$migration"
done
echo "Migrations complete."
