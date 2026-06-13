#!/usr/bin/env bash
# Deploy status-page/ to Vercel when VERCEL_* secrets are configured.
set -euo pipefail

if [ -z "${VERCEL_TOKEN:-}" ] || [ -z "${VERCEL_ORG_ID:-}" ] || [ -z "${VERCEL_PROJECT_ID:-}" ]; then
  echo "Vercel secrets not configured; skipping status page deploy."
  echo "Set VERCEL_TOKEN, VERCEL_ORG_ID, and VERCEL_PROJECT_ID to enable automated deploys."
  exit 0
fi

cd status-page
if command -v vercel >/dev/null 2>&1; then
  vercel deploy --prod --token "$VERCEL_TOKEN"
else
  npx --yes vercel@latest deploy --prod --token "$VERCEL_TOKEN"
fi
