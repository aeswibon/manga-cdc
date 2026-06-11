#!/usr/bin/env bash
# Legacy helper — the scraper applies migrations automatically on startup via goose.
# This script remains for manual/one-off use outside the scraper container.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL is required}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$ROOT/db/migrations}"

if ! command -v goose >/dev/null 2>&1; then
  echo "error: goose CLI not installed; migrations run automatically when the scraper starts" >&2
  echo "install: go install github.com/pressly/goose/v3/cmd/goose@latest" >&2
  exit 1
fi

echo "Applying database migrations from ${MIGRATIONS_DIR}..."
goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" up
echo "Migrations complete."
