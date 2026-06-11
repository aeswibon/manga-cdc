#!/usr/bin/env bash
set -euo pipefail

REPO_DIR="${MANGA_CDC_DIR:-$HOME/manga-cdc}"
OBSERVABILITY_REQUIRED="${OBSERVABILITY_REQUIRED:-true}"
cd "$REPO_DIR"

failures=0

observability_mode() {
  grep -E '^OBSERVABILITY_MODE=' .env | cut -d= -f2- | tr -d '\r' || true
}

OBSERVABILITY_MODE="$(observability_mode)"
OBSERVABILITY_MODE="${OBSERVABILITY_MODE:-grafana-cloud}"

compose() {
  local args=(docker compose --env-file .env -f docker-compose.prod.yml)
  case "$OBSERVABILITY_MODE" in
    grafana-cloud)
      args+=(-f docker-compose.observability-cloud.yml)
      ;;
    self-hosted)
      args+=(-f docker-compose.observability.yml)
      ;;
  esac
  "${args[@]}" "$@"
}

service_running() {
  compose ps --status running --services | grep -qx "$1"
}

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

warn() {
  local label="$1"
  shift
  if "$@"; then
    echo "OK: $label"
  else
    echo "WARN: $label" >&2
  fi
}

echo "=== observability mode: ${OBSERVABILITY_MODE} ==="
echo '=== Container Status ==='
compose ps

echo '=== Image tags ==='
grep -E '^(SCRAPER_IMAGE|NOTIFICATION_IMAGE)=' .env

check "scraper container running" service_running scraper
check "notification-service container running" service_running notification-service
check "discord webhook configured" grep -q '^DISCORD_WEBHOOK_URL=https' .env
check "cdc enabled" grep -q '^CDC_ENABLED=true' .env
check "scraper healthz" curl -sf --max-time 15 http://127.0.0.1:2112/healthz
check "scraper readyz" curl -sf --max-time 15 http://127.0.0.1:2112/readyz
check "scraper metrics present" bash -c 'curl -sf --max-time 15 http://127.0.0.1:2112/metrics | grep -q "^scraper_chapters_"'
check "notification health" curl -sf --max-time 15 http://127.0.0.1:8080/actuator/health

warn "notification logs API" curl -sf --max-time 15 "http://127.0.0.1:8080/api/logs?limit=5" >/dev/null

if [ "$OBSERVABILITY_REQUIRED" = "true" ] && [ "$OBSERVABILITY_MODE" != "off" ] && [ "$OBSERVABILITY_MODE" != "disabled" ] && [ "$OBSERVABILITY_MODE" != "false" ]; then
  case "$OBSERVABILITY_MODE" in
    grafana-cloud)
      check "alloy container running" service_running alloy
      check "grafana cloud remote_write configured" grep -q '^GRAFANA_CLOUD_PROMETHEUS_URL=https' .env
      check "alloy metrics endpoint" curl -sf --max-time 15 http://127.0.0.1:12345/metrics >/dev/null
      ;;
    self-hosted)
      check "prometheus container running" service_running prometheus
      check "grafana container running" service_running grafana
      check "prometheus healthy" curl -sf --max-time 15 http://127.0.0.1:9090/-/healthy
      check "grafana healthy" curl -sf --max-time 15 http://127.0.0.1:3000/api/health
      check "grafana dashboard provisioned" bash -c \
        'curl -sf --max-time 15 "http://127.0.0.1:3000/api/search?type=dash-db" | grep -q manga-cdc'
      ;;
  esac
fi

if [ "$failures" -gt 0 ]; then
  echo "verification failed with $failures required check(s)" >&2
  exit 1
fi

echo "all required verification checks passed"
