#!/usr/bin/env bash
# Deploy status-page/ to Vercel when VERCEL_* secrets are configured.
set -euo pipefail

PROJECT_ID="${VERCEL_STATUS_PAGE_PROJECT_ID:-${VERCEL_PROJECT_ID:-}}"

if [ -z "${VERCEL_TOKEN:-}" ] || [ -z "${VERCEL_ORG_ID:-}" ] || [ -z "$PROJECT_ID" ]; then
  echo "Vercel secrets not configured; skipping status page deploy."
  echo "Set VERCEL_TOKEN, VERCEL_ORG_ID, and VERCEL_STATUS_PAGE_PROJECT_ID (or VERCEL_PROJECT_ID)."
  exit 0
fi

export VERCEL_ORG_ID
export VERCEL_PROJECT_ID="$PROJECT_ID"

cd status-page
if command -v vercel >/dev/null 2>&1; then
  vercel deploy --prod --yes --token "$VERCEL_TOKEN"
else
  npx --yes vercel@latest deploy --prod --yes --token "$VERCEL_TOKEN"
fi
