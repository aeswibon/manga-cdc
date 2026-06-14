#!/usr/bin/env bash
# Read-only production security smoke checks (requires API_READ_KEY in env).
set -euo pipefail

NOTIFIER_URL="${NOTIFIER_URL:-https://manga-cdc-notifier-prod-uzvgkbgfpq-uc.a.run.app}"
STATUS_URL="${STATUS_URL:-https://manga-cdc-status.vercel.app}"
API_READ_KEY="${API_READ_KEY:?Set API_READ_KEY to run production security checks}"

failures=0

check() {
  local label="$1"
  shift
  if "$@"; then
    echo "OK: $label"
  else
    echo "FAIL: $label" >&2
    failures=$((failures + 1))
  fi
}

echo "=== Production security checks ==="
echo "Notifier: $NOTIFIER_URL"
echo "Status:   $STATUS_URL"

check "actuator health is public" \
  curl -sf --max-time 15 "$NOTIFIER_URL/actuator/health" | grep -q '"status"'

check "read API rejects missing key" \
  bash -c 'code=$(curl -sS -o /dev/null -w "%{http_code}" --max-time 15 "'"$NOTIFIER_URL"'/api/stats"); test "$code" = "401"'

check "read API accepts API key" \
  curl -sf --max-time 15 -H "X-Api-Key: $API_READ_KEY" "$NOTIFIER_URL/api/stats" | grep -q '"total_series"'

check "pipeline health requires API key" \
  bash -c 'code=$(curl -sS -o /dev/null -w "%{http_code}" --max-time 15 "'"$NOTIFIER_URL"'/api/pipeline/health"); test "$code" = "401"'

check "pipeline health accepts API key" \
  curl -sf --max-time 15 -H "X-Api-Key: $API_READ_KEY" "$NOTIFIER_URL/api/pipeline/health" | grep -q '"status"'

check "webhook rejects unsigned POST" \
  bash -c 'code=$(curl -sS -o /dev/null -w "%{http_code}" --max-time 15 -X POST -H "Content-Type: application/json" -d "{\"op\":\"c\"}" "'"$NOTIFIER_URL"'/api/webhook"); test "$code" = "401"'

check "status page is public" \
  curl -sf --max-time 15 "$STATUS_URL/api/status" | grep -q '"status"'

check "cover proxy rejects unknown host" \
  bash -c 'code=$(curl -sS -o /dev/null -w "%{http_code}" --max-time 15 "'"${DASHBOARD_URL:-https://manga-cdc.vercel.app}"'/api/cover?url=https://example.com/x.png"); test "$code" = "400"'

if [ "$failures" -gt 0 ]; then
  echo "=== $failures check(s) failed ===" >&2
  exit 1
fi

echo "=== All production security checks passed ==="
