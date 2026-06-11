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

observability_mode() {
  grep -E '^OBSERVABILITY_MODE=' .env | cut -d= -f2- | tr -d '\r' || true
}

OBSERVABILITY_MODE="$(observability_mode)"
OBSERVABILITY_MODE="${OBSERVABILITY_MODE:-grafana-cloud}"

export COMPOSE_HTTP_TIMEOUT="${COMPOSE_HTTP_TIMEOUT:-300}"
export DOCKER_CLIENT_TIMEOUT="${DOCKER_CLIENT_TIMEOUT:-300}"

compose=(docker compose --env-file .env -f docker-compose.prod.yml)

case "$OBSERVABILITY_MODE" in
  grafana-cloud)
    if [ ! -f docker-compose.observability-cloud.yml ]; then
      echo "error: OBSERVABILITY_MODE=grafana-cloud but docker-compose.observability-cloud.yml is missing" >&2
      exit 1
    fi
    compose+=(-f docker-compose.observability-cloud.yml)
    echo "observability: grafana-cloud (Alloy remote_write)"
  ;;
  self-hosted)
    if [ ! -f docker-compose.observability.yml ]; then
      echo "error: OBSERVABILITY_MODE=self-hosted but docker-compose.observability.yml is missing" >&2
      exit 1
    fi
    compose+=(-f docker-compose.observability.yml)
    echo "observability: self-hosted (Prometheus + Grafana on VM)"
  ;;
  off|disabled|false)
    echo "observability: disabled"
  ;;
  *)
    echo "error: unknown OBSERVABILITY_MODE=$OBSERVABILITY_MODE (use grafana-cloud, self-hosted, or off)" >&2
    exit 1
  ;;
esac

echo "=== pull images (timeout ${PULL_TIMEOUT_SECONDS}s) ==="
timeout "${PULL_TIMEOUT_SECONDS}" "${compose[@]}" pull --quiet

echo "=== start services (timeout ${UP_TIMEOUT_SECONDS}s) ==="
timeout "${UP_TIMEOUT_SECONDS}" "${compose[@]}" up -d --remove-orphans

echo "=== container status ==="
"${compose[@]}" ps

case "$OBSERVABILITY_MODE" in
  grafana-cloud)
    stack_url="$(grep -E '^GRAFANA_CLOUD_STACK_URL=' .env | cut -d= -f2- | tr -d '\r' || true)"
    if [ -n "$stack_url" ]; then
      echo "Grafana Cloud: ${stack_url}/d/manga-cdc-overview/manga-cdc"
    else
      echo "Grafana Cloud: set GRAFANA_CLOUD_STACK_URL in .env for dashboard link"
    fi
  ;;
  self-hosted)
    VM_IP="$(curl -sf --max-time 5 -H 'Metadata-Flavor: Google' \
      http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip 2>/dev/null || true)"
    if [ -n "$VM_IP" ]; then
      echo "Grafana:    http://${VM_IP}:3000/d/manga-cdc-overview/manga-cdc"
      echo "Prometheus: http://${VM_IP}:9090"
    fi
  ;;
esac
