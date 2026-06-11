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
USE_IAP="${USE_IAP:-auto}"

ssh_args=(--zone "${ZONE}" --quiet)
case "$USE_IAP" in
  true|1|yes) ssh_args+=(--tunnel-through-iap) ;;
  false|0|no) ;;
  auto)
    if gcloud compute instances describe "${INSTANCE}" --zone "${ZONE}" \
      --format='get(networkInterfaces[0].accessConfigs[0].natIP)' 2>/dev/null | grep -q .; then
      echo "SSH: using public IP (set USE_IAP=true to force IAP)"
    else
      ssh_args+=(--tunnel-through-iap)
      echo "SSH: no public IP, using IAP"
    fi
    ;;
esac

gcloud_ssh() {
  gcloud compute ssh "${SSH_USER}@${INSTANCE}" "${ssh_args[@]}" "$@"
}

echo "=== sync manga-cdc on ${INSTANCE} (${ZONE}) ==="
gcloud_ssh \
  --command "set -euo pipefail
    mkdir -p ~/manga-cdc
    cd ~/manga-cdc
    if [ ! -d .git ]; then git clone '${REPO_URL}' .; fi
    git fetch origin --force --tags
    git reset --hard 'origin/${REF}'
    chmod +x scripts/enable-observability-on-vm.sh scripts/verify-prod-on-vm.sh
  "

echo "=== enable observability stack ==="
gcloud_ssh --command "bash ~/manga-cdc/scripts/enable-observability-on-vm.sh"

echo "=== verify ==="
gcloud_ssh --command "OBSERVABILITY_REQUIRED=true bash ~/manga-cdc/scripts/verify-prod-on-vm.sh" || true

EXTERNAL_IP="$(gcloud compute instances describe "${INSTANCE}" --zone "${ZONE}" \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)')"
echo ""
echo "Grafana dashboard: http://${EXTERNAL_IP}:3000/d/manga-cdc-overview/manga-cdc"
echo "Prometheus:        http://${EXTERNAL_IP}:9090"
