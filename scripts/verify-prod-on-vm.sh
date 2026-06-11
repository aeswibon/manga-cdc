#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${MANGA_CDC_DIR:-$HOME/manga-cdc}"
cd "$REPO_DIR"

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

echo '=== Container Status ==='
docker compose --env-file .env -f docker-compose.prod.yml ps

echo '=== Image tags ==='
grep -E '^(SCRAPER_IMAGE|NOTIFICATION_IMAGE)=' .env

check "scraper container running" docker compose --env-file .env -f docker-compose.prod.yml ps --status running --services | grep -qx scraper
check "notification-service container running" docker compose --env-file .env -f docker-compose.prod.yml ps --status running --services | grep -qx notification-service
check "discord webhook configured" grep -q '^DISCORD_WEBHOOK_URL=https' .env
check "cdc enabled" grep -q '^CDC_ENABLED=true' .env
check "scraper healthz" curl -sf --max-time 15 http://127.0.0.1:2112/healthz
check "scraper readyz" curl -sf --max-time 15 http://127.0.0.1:2112/readyz
check "scraper metrics present" bash -c 'curl -sf --max-time 15 http://127.0.0.1:2112/metrics | grep -q "^scraper_chapters_"'
check "notification health" curl -sf --max-time 15 http://127.0.0.1:8080/actuator/health

echo '=== Notification logs API ==='
if curl -sf --max-time 15 "http://127.0.0.1:8080/api/logs?limit=5" >/dev/null; then
  echo "OK: notification logs API reachable"
else
  echo "WARN: notification logs API unavailable (requires notification-service >= v0.3.0)" >&2
fi

if [ "$failures" -gt 0 ]; then
  echo "verification failed with $failures required check(s)" >&2
  exit 1
fi

echo "all required verification checks passed"
