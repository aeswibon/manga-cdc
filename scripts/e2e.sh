#!/usr/bin/env bash
set -euo pipefail

# Full stack end-to-end test for manga-cdc (local Kafka via Redpanda, direct publish)
# Prerequisites: Docker
#
# Usage:
#   ./scripts/e2e.sh
#
# CI sets SCRAPER_IMAGE / NOTIFICATION_IMAGE from the docker-snapshot job.

cd "$(dirname "$0")/.."
ROOT=$(pwd)
LOG="$ROOT/target/e2e.log"
CI_SHA="$(git rev-parse HEAD)"
TOPIC="mangacdc.public.chapters"
SERIES_ID="00000000-0000-0000-0000-000000000001"
CHAPTER_ID="00000000-0000-0000-0000-000000000101"

mkdir -p target
echo "=== manga-cdc E2E Test ===" | tee "$LOG"

cleanup() {
  echo "Cleaning up..." | tee -a "$LOG"
  cd "$ROOT"
  docker compose down -v 2>/dev/null || true
}
trap cleanup EXIT

if [ "${GITHUB_ACTIONS:-}" = "true" ]; then
  SCRAPER_IMAGE="${SCRAPER_IMAGE:-ghcr.io/aeswibon/manga-cdc/scraper:ci-${CI_SHA}}"
  NOTIFICATION_IMAGE="${NOTIFICATION_IMAGE:-ghcr.io/aeswibon/manga-cdc/notification-service:ci-${CI_SHA}}"
  export SCRAPER_IMAGE NOTIFICATION_IMAGE
  echo "CI images: scraper=${SCRAPER_IMAGE}, notification=${NOTIFICATION_IMAGE}" | tee -a "$LOG"
fi

echo "Starting Postgres and Redpanda..." | tee -a "$LOG"
docker compose up -d --remove-orphans postgres redpanda 2>&1 | tee -a "$LOG"

echo "Waiting for services to be healthy..." | tee -a "$LOG"
postgres_ready=false
for _ in $(seq 1 30); do
  if docker compose exec postgres pg_isready -U mangacdc >/dev/null 2>&1; then
    echo "PostgreSQL ready" | tee -a "$LOG"
    postgres_ready=true
    break
  fi
  sleep 2
done
if [ "$postgres_ready" != "true" ]; then
  echo "FAIL: PostgreSQL not ready within timeout" | tee -a "$LOG"
  exit 1
fi

redpanda_ready=false
for _ in $(seq 1 60); do
  if docker compose exec redpanda rpk cluster health 2>/dev/null | grep -q 'Healthy:[[:space:]]*true'; then
    echo "Redpanda ready" | tee -a "$LOG"
    redpanda_ready=true
    break
  fi
  sleep 2
done
if [ "$redpanda_ready" != "true" ]; then
  echo "FAIL: Redpanda not ready within timeout" | tee -a "$LOG"
  exit 1
fi

echo "Creating Kafka topic ${TOPIC}..." | tee -a "$LOG"
docker compose exec -T redpanda rpk topic create "$TOPIC" 2>&1 | tee -a "$LOG" || true

echo "Setting up test data..." | tee -a "$LOG"
seed_sql=$(cat <<SQL
DELETE FROM notification_logs WHERE chapter_id = '${CHAPTER_ID}';
DELETE FROM chapters WHERE series_id = '${SERIES_ID}' OR id = '${CHAPTER_ID}';
DELETE FROM manga_series WHERE source_id = 'e2e-source' OR id = '${SERIES_ID}';

INSERT INTO manga_series (id, source_id, title, source_url, status, is_active)
VALUES ('${SERIES_ID}', 'e2e-source', 'E2E Test Series', 'https://example.com/e2e', 'ONGOING', true);

INSERT INTO chapters (id, series_id, chapter_num, title, url, is_new)
VALUES ('${CHAPTER_ID}', '${SERIES_ID}', 1, 'Chapter 1', 'https://example.com/e2e/ch-1', true);
SQL
)
seeded=false
for _ in $(seq 1 15); do
  if printf '%s\n' "$seed_sql" | docker compose exec -T postgres psql -U mangacdc -d mangacdc >>"$LOG" 2>&1; then
    seeded=true
    break
  fi
  sleep 2
done
if [ "$seeded" != "true" ]; then
  echo "FAIL: could not seed Postgres test data" | tee -a "$LOG"
  exit 1
fi

chapter_count=$(docker compose exec -T postgres psql -U mangacdc -d mangacdc -tAc \
  "SELECT COUNT(*) FROM chapters WHERE id = '${CHAPTER_ID}'" 2>/dev/null | tr -d '[:space:]')
if [ "${chapter_count:-0}" != "1" ]; then
  echo "FAIL: expected seeded chapter ${CHAPTER_ID}, found count=${chapter_count:-0}" | tee -a "$LOG"
  exit 1
fi

echo "Publishing chapter event to Kafka (scraper format)..." | tee -a "$LOG"
EVENT=$(cat <<EOF
{"op":"c","after":{"id":"${CHAPTER_ID}","series_id":"${SERIES_ID}","chapter_num":1,"title":"Chapter 1","url":"https://example.com/e2e/ch-1","is_new":true}}
EOF
)
echo "$EVENT" | docker compose exec -T redpanda rpk topic produce "$TOPIC" -k "$CHAPTER_ID" 2>&1 | tee -a "$LOG"

echo "Starting notification service..." | tee -a "$LOG"
if [ "${GITHUB_ACTIONS:-}" = "true" ]; then
  if ! docker image inspect "${NOTIFICATION_IMAGE}" >/dev/null 2>&1; then
    echo "Pulling ${NOTIFICATION_IMAGE}..." | tee -a "$LOG"
    docker pull "${NOTIFICATION_IMAGE}" 2>&1 | tee -a "$LOG"
  fi
  DISCORD_WEBHOOK_URL=http://127.0.0.1:9/unreachable \
  CDC_ENABLED=true \
  docker compose up -d --no-build notification-service 2>&1 | tee -a "$LOG"
else
  DISCORD_WEBHOOK_URL=http://127.0.0.1:9/unreachable \
  CDC_ENABLED=true \
  docker compose up -d --build notification-service 2>&1 | tee -a "$LOG"
fi

echo "Waiting for notification consumer..." | tee -a "$LOG"
for _ in $(seq 1 60); do
  STATUS=$(docker compose exec -T postgres psql -U mangacdc -d mangacdc -tAc \
    "SELECT status FROM notification_logs WHERE channel='discord' AND chapter_id='${CHAPTER_ID}'" 2>/dev/null | tr -d '[:space:]')
  if [ "$STATUS" = "SENT" ] || [ "$STATUS" = "FAILED" ]; then
    echo "PASS: Notification logged (status=$STATUS)" | tee -a "$LOG"
    echo "" | tee -a "$LOG"
    echo "=== E2E Test Complete ===" | tee -a "$LOG"
    exit 0
  fi
  sleep 2
done

echo "FAIL: No notification log found within timeout" | tee -a "$LOG"
docker compose logs --tail=50 notification-service 2>&1 | tee -a "$LOG" || true
exit 1
