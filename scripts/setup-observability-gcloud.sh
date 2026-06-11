#!/usr/bin/env bash
# Enable Grafana + Prometheus on the prod GCP VM via gcloud SSH.
#
# Usage:
#   ./scripts/setup-observability-gcloud.sh
#   ./scripts/setup-observability-gcloud.sh mangacdc-vm us-east1-b github-actions-cd
#
# Requires: gcloud auth, SSH access to the VM (OS Login or deploy SSH key).
set -euo pipefail

INSTANCE="${1:-mangacdc-vm}"
ZONE="${2:-us-east1-b}"
SSH_USER="${3:-}"

if [ -z "$SSH_USER" ]; then
  SSH_USER="$(gcloud config get-value account 2>/dev/null | tr '@' '_' | tr '.' '_' | tr '[:upper:]' '[:lower:]' || true)"
fi
if [ -z "$SSH_USER" ] || [ "$SSH_USER" = "(unset)" ]; then
  echo "error: set SSH user as 3rd arg or configure gcloud account" >&2
  exit 1
fi

REPO_URL="${MANGA_CDC_REPO_URL:-https://github.com/aeswibon/manga-cdc.git}"
REF="${MANGA_CDC_REF:-master}"

echo "=== sync manga-cdc on ${INSTANCE} (${ZONE}) ==="
gcloud compute ssh "${SSH_USER}@${INSTANCE}" \
  --zone "${ZONE}" \
  --tunnel-through-iap \
  --command "set -euo pipefail
    mkdir -p ~/manga-cdc
    cd ~/manga-cdc
    if [ ! -d .git ]; then git clone '${REPO_URL}' .; fi
    git fetch origin --force --tags
    git reset --hard 'origin/${REF}'
    chmod +x scripts/enable-observability-on-vm.sh scripts/verify-prod-on-vm.sh
  "

echo "=== enable observability stack ==="
gcloud compute ssh "${SSH_USER}@${INSTANCE}" \
  --zone "${ZONE}" \
  --tunnel-through-iap \
  --command "bash ~/manga-cdc/scripts/enable-observability-on-vm.sh"

echo "=== verify ==="
gcloud compute ssh "${SSH_USER}@${INSTANCE}" \
  --zone "${ZONE}" \
  --tunnel-through-iap \
  --command "OBSERVABILITY_REQUIRED=true bash ~/manga-cdc/scripts/verify-prod-on-vm.sh" || true

EXTERNAL_IP="$(gcloud compute instances describe "${INSTANCE}" --zone "${ZONE}" \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)')"
echo ""
echo "Grafana dashboard: http://${EXTERNAL_IP}:3000/d/manga-cdc-overview/manga-cdc"
echo "Prometheus:        http://${EXTERNAL_IP}:9090"
