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

echo ""
echo "Security Note: Grafana and Prometheus ports are bound to 127.0.0.1."
echo "To access them securely from your local browser, establish an SSH tunnel or use GCP IAP TCP forwarding:"
echo ""
echo "For Grafana (port 3000):"
echo "  gcloud compute start-iap-tunnel <vm-instance-name> 3000 --local-host-port=localhost:3000 --zone=<zone>"
echo "  Then open: http://localhost:3000/d/manga-cdc-overview/manga-cdc"
echo ""
echo "For Prometheus (port 9090):"
echo "  gcloud compute start-iap-tunnel <vm-instance-name> 9090 --local-host-port=localhost:9090 --zone=<zone>"
echo "  Then open: http://localhost:9090"
