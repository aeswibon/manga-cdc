#!/usr/bin/env bash
# gcloud compute scp wrapper for CI.
set -euo pipefail

: "${GCP_SSH_USER:?}"
: "${GCP_VM_NAME:?}"
: "${GCP_ZONE:?}"
: "${GCP_SSH_KEY_FILE:?}"
: "${1:?local path required}"
: "${2:?remote path required}"

target="${GCP_SSH_USER}@${GCP_VM_NAME}"

gcloud compute scp "$1" "${target}:$2" \
  --zone "$GCP_ZONE" \
  --ssh-key-file "$GCP_SSH_KEY_FILE" \
  --force-key-file-overwrite \
  --quiet \
  --scp-flag="-o ConnectTimeout=30" \
  --scp-flag="-o ServerAliveInterval=15"
