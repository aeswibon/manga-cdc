#!/usr/bin/env bash
# gcloud compute ssh wrapper for CI (avoid --ssh-flag gcloud parsing bugs).
set -euo pipefail

: "${GCP_SSH_USER:?}"
: "${GCP_VM_NAME:?}"
: "${GCP_ZONE:?}"
: "${GCP_SSH_KEY_FILE:?}"

target="${GCP_SSH_USER}@${GCP_VM_NAME}"
ssh_opts=(-o ServerAliveInterval=15 -o ServerAliveCountMax=40 -o ConnectTimeout=30 -o TCPKeepAlive=yes)

args=(
  gcloud compute ssh "$target"
  --zone "$GCP_ZONE"
  --ssh-key-file "$GCP_SSH_KEY_FILE"
  --force-key-file-overwrite
  --quiet
)

if [ -n "${1:-}" ]; then
  args+=(--command "$1")
fi

args+=(-- "${ssh_opts[@]}")
"${args[@]}"
