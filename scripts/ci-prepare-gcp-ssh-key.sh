#!/usr/bin/env bash
# Write multiline deploy SSH key from env (GitHub Actions safe).
set -euo pipefail

: "${GCP_SSH_PRIVATE_KEY:?GCP_SSH_PRIVATE_KEY is required}"
: "${RUNNER_TEMP:?RUNNER_TEMP is required}"

key="${RUNNER_TEMP}/gcp_ssh_key"
umask 077
printf '%s\n' "$GCP_SSH_PRIVATE_KEY" > "$key"
chmod 600 "$key"
ssh-keygen -y -f "$key" > "${key}.pub"
chmod 644 "${key}.pub"
echo "GCP_SSH_KEY_FILE=$key" >> "${GITHUB_ENV:?GITHUB_ENV is required}"
