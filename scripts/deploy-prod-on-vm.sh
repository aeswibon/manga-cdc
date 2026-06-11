#!/usr/bin/env bash
# Run on the GCP VM after repo sync and .env have been prepared.
set -euo pipefail

REPO_DIR="${MANGA_CDC_DIR:-$HOME/manga-cdc}"
PULL_TIMEOUT_SECONDS="${PULL_TIMEOUT_SECONDS:-900}"
UP_TIMEOUT_SECONDS="${UP_TIMEOUT_SECONDS:-180}"

cd "$REPO_DIR"

if [ ! -f .env ]; then
  echo "error: $REPO_DIR/.env not found (expected from deploy scp step)" >&2
  exit 1
fi

if [ ! -w .env ]; then
  sudo rm -f .env
  echo "error: .env was not writable; re-run deploy to copy a fresh file" >&2
  exit 1
fi

chmod 600 .env

export COMPOSE_HTTP_TIMEOUT="${COMPOSE_HTTP_TIMEOUT:-300}"
export DOCKER_CLIENT_TIMEOUT="${DOCKER_CLIENT_TIMEOUT:-300}"

echo "=== pull images (timeout ${PULL_TIMEOUT_SECONDS}s) ==="
timeout "${PULL_TIMEOUT_SECONDS}" docker compose --env-file .env -f docker-compose.prod.yml pull --quiet

echo "=== start services (timeout ${UP_TIMEOUT_SECONDS}s) ==="
timeout "${UP_TIMEOUT_SECONDS}" docker compose --env-file .env -f docker-compose.prod.yml up -d --remove-orphans

echo "=== container status ==="
docker compose --env-file .env -f docker-compose.prod.yml ps
