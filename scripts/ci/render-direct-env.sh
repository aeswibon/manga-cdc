#!/usr/bin/env bash
# Render prod.env for VM direct deploys.
set -euo pipefail

: "${RUNNER_TEMP:?RUNNER_TEMP is required}"
: "${SCRAPER_IMAGE:?}"
: "${NOTIFICATION_IMAGE:?}"
: "${DATABASE_URL:?}"

chmod +x scripts/render-prod-env.sh
./scripts/render-prod-env.sh > "${RUNNER_TEMP}/prod.env"
chmod 600 "${RUNNER_TEMP}/prod.env"
