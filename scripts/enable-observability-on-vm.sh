#!/usr/bin/env bash
# Start Prometheus + Grafana on the VM (expects manga-cdc app already deployed).
set -euo pipefail

REPO_DIR="${MANGA_CDC_DIR:-$HOME/manga-cdc}"
PULL_TIMEOUT_SECONDS="${PULL_TIMEOUT_SECONDS:-300}"
UP_TIMEOUT_SECONDS="${UP_TIMEOUT_SECONDS:-120}"

cd "$REPO_DIR"

if [ ! -f docker-compose.observability.yml ]; then
  echo "error: $REPO_DIR missing docker-compose.observability.yml — git pull first" >&2
  exit 1
fi

export COMPOSE_HTTP_TIMEOUT="${COMPOSE_HTTP_TIMEOUT:-300}"
export DOCKER_CLIENT_TIMEOUT="${DOCKER_CLIENT_TIMEOUT:-300}"

compose=(docker compose)
if [ -f .env ]; then
  compose+=(--env-file .env)
fi
compose+=(
  -f docker-compose.prod.yml
  -f docker-compose.observability.yml
)

echo "=== pull observability images (timeout ${PULL_TIMEOUT_SECONDS}s) ==="
timeout "${PULL_TIMEOUT_SECONDS}" "${compose[@]}" pull prometheus grafana --quiet

echo "=== start observability stack (timeout ${UP_TIMEOUT_SECONDS}s) ==="
timeout "${UP_TIMEOUT_SECONDS}" "${compose[@]}" up -d prometheus grafana

echo "=== observability status ==="
"${compose[@]}" ps prometheus grafana

VM_IP="$(curl -sf --max-time 5 -H 'Metadata-Flavor: Google' http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip 2>/dev/null || true)"
if [ -n "$VM_IP" ]; then
  echo "Grafana:    http://${VM_IP}:3000/d/manga-cdc-overview/manga-cdc"
  echo "Prometheus: http://${VM_IP}:9090"
else
  echo "Grafana:    http://<vm-external-ip>:3000/d/manga-cdc-overview/manga-cdc"
  echo "Prometheus: http://<vm-external-ip>:9090"
fi
