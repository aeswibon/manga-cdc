#!/usr/bin/env bash
# Deploy dashboard/ to Vercel when VERCEL_* secrets are configured.
set -euo pipefail

if [ -z "${VERCEL_TOKEN:-}" ] || [ -z "${VERCEL_ORG_ID:-}" ] || [ -z "${VERCEL_DASHBOARD_PROJECT_ID:-}" ]; then
  echo "Vercel dashboard secrets not configured; skipping dashboard deploy."
  echo "Set VERCEL_TOKEN, VERCEL_ORG_ID, and VERCEL_DASHBOARD_PROJECT_ID to enable automated deploys."
  exit 0
fi

export VERCEL_PROJECT_ID="$VERCEL_DASHBOARD_PROJECT_ID"

cd dashboard
if command -v vercel >/dev/null 2>&1; then
  vercel deploy --prod --token "$VERCEL_TOKEN"
else
  npx --yes vercel@latest deploy --prod --token "$VERCEL_TOKEN"
fi
