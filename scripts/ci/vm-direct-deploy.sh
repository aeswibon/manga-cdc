#!/usr/bin/env bash
# Package and deploy docker-compose stack to a generic VM over SSH.
set -euo pipefail

: "${RUNNER_TEMP:?}"
: "${SSH_KEY:?}"
: "${VM_USER:?}"
: "${VM_HOST:?}"

files_to_pack=(
  docker-compose.prod.yml
  docker-compose.observability-cloud.yml
  docker-compose.observability.yml
  alloy
  config/prometheus.yml
  grafana
  scripts/deploy-prod-on-vm.sh
  scripts/verify-prod-on-vm.sh
)
if [ -f Caddyfile ]; then
  files_to_pack+=(Caddyfile)
fi
tar -czf "${RUNNER_TEMP}/deploy.tar.gz" "${files_to_pack[@]}"

mkdir -p ~/.ssh
echo "$SSH_KEY" > ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa

ssh_opts=(-i ~/.ssh/id_rsa)
if [ -n "${VM_SSH_KNOWN_HOSTS:-}" ]; then
  known_hosts_file="${RUNNER_TEMP}/known_hosts"
  printf '%s\n' "$VM_SSH_KNOWN_HOSTS" > "$known_hosts_file"
  ssh_opts+=(-o StrictHostKeyChecking=yes -o UserKnownHostsFile="$known_hosts_file")
else
  ssh_opts+=(-o StrictHostKeyChecking=accept-new)
fi

scp "${ssh_opts[@]}" "${RUNNER_TEMP}/deploy.tar.gz" "${VM_USER}@${VM_HOST}:~/deploy.tar.gz"
scp "${ssh_opts[@]}" "${RUNNER_TEMP}/prod.env" "${VM_USER}@${VM_HOST}:~/prod.env"

ssh "${ssh_opts[@]}" "${VM_USER}@${VM_HOST}" \
  "mkdir -p ~/manga-cdc; tar -xzf ~/deploy.tar.gz -C ~/manga-cdc; mv ~/prod.env ~/manga-cdc/.env; chmod +x ~/manga-cdc/scripts/deploy-prod-on-vm.sh ~/manga-cdc/scripts/verify-prod-on-vm.sh"
ssh "${ssh_opts[@]}" "${VM_USER}@${VM_HOST}" "bash ~/manga-cdc/scripts/deploy-prod-on-vm.sh"
ssh "${ssh_opts[@]}" "${VM_USER}@${VM_HOST}" "OBSERVABILITY_REQUIRED=true bash ~/manga-cdc/scripts/verify-prod-on-vm.sh"
