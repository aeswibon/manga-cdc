#!/usr/bin/env bash
set -euo pipefail

# Full stack end-to-end test for manga-cdc (local Kafka via Redpanda, direct publish)
# Prerequisites: Docker, Java 21+, jq
#
# Usage:
#   ./scripts/e2e.sh              # run full E2E test
#   ./scripts/e2e.sh --skip-build # skip Java build

SKIP_BUILD="${1:-}"

cd "$(dirname "$0")/.."
ROOT=$(pwd)
LOG="$ROOT/target/e2e.log"
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

if [ "$SKIP_BUILD" != "--skip-build" ]; then
  echo "Building notification service..." | tee -a "$LOG"
  (cd notification-service && mvn -q package -DskipTests) 2>&1 | tee -a "$LOG"
fi

echo "Starting Postgres and Redpanda..." | tee -a "$LOG"
docker compose up -d postgres redpanda 2>&1 | tee -a "$LOG"

echo "Waiting for services to be healthy..." | tee -a "$LOG"
for _ in $(seq 1 30); do
  if docker compose exec postgres pg_isready -U mangacdc >/dev/null 2>&1; then
    echo "PostgreSQL ready" | tee -a "$LOG"
    break
  fi
  sleep 2
done

for _ in $(seq 1 30); do
  if docker compose exec redpanda rpk cluster info 2>/dev/null | grep -q leader_id; then
    echo "Redpanda ready" | tee -a "$LOG"
    break
  fi
  sleep 2
done

echo "Setting up test data..." | tee -a "$LOG"
docker compose exec -T postgres psql -U mangacdc -d mangacdc <<SQL
  INSERT INTO manga_series (id, source_id, title, source_url, status, is_active)
  VALUES ('${SERIES_ID}', 'e2e-source', 'E2E Test Series', 'https://example.com/e2e', 'ONGOING', true)
  ON CONFLICT (source_id) DO NOTHING;

  INSERT INTO chapters (id, series_id, chapter_num, title, url, is_new)
  VALUES ('${CHAPTER_ID}', '${SERIES_ID}', 1, 'Chapter 1', 'https://example.com/e2e/ch-1', true)
  ON CONFLICT (series_id, chapter_num) DO NOTHING;
SQL

echo "Publishing chapter event to Kafka (scraper format)..." | tee -a "$LOG"
EVENT=$(cat <<EOF
{"op":"c","after":{"id":"${CHAPTER_ID}","series_id":"${SERIES_ID}","chapter_num":1,"title":"Chapter 1","url":"https://example.com/e2e/ch-1","is_new":true}}
EOF
)
echo "$EVENT" | docker compose exec -T redpanda rpk topic produce "$TOPIC" -k "$CHAPTER_ID" 2>&1 | tee -a "$LOG"

echo "Starting notification service..." | tee -a "$LOG"
CDC_ENABLED=true \
SPRING_KAFKA_BOOTSTRAP_SERVERS=localhost:9092 \
SPRING_DATASOURCE_URL=jdbc:postgresql://localhost:5432/mangacdc \
SPRING_DATASOURCE_USERNAME=mangacdc \
SPRING_DATASOURCE_PASSWORD=mangacdc \
DISCORD_WEBHOOK_URL="http://localhost:9999/mock" \
java -jar notification-service/target/notification-service-0.0.1-SNAPSHOT.jar >>"$LOG" 2>&1 &
NOTIFY_PID=$!

echo "Waiting for notification consumer..." | tee -a "$LOG"
for _ in $(seq 1 30); do
  STATUS=$(docker compose exec -T postgres psql -U mangacdc -d mangacdc -tAc \
    "SELECT status FROM notification_logs WHERE channel='discord' AND chapter_id='${CHAPTER_ID}'" 2>/dev/null | tr -d '[:space:]')
  if [ "$STATUS" = "SENT" ] || [ "$STATUS" = "FAILED" ]; then
    echo "PASS: Notification logged (status=$STATUS)" | tee -a "$LOG"
    kill "$NOTIFY_PID" 2>/dev/null || true
    wait "$NOTIFY_PID" 2>/dev/null || true
    echo "" | tee -a "$LOG"
    echo "=== E2E Test Complete ===" | tee -a "$LOG"
    exit 0
  fi
  sleep 2
done

echo "FAIL: No notification log found within timeout" | tee -a "$LOG"
kill "$NOTIFY_PID" 2>/dev/null || true
wait "$NOTIFY_PID" 2>/dev/null || true
exit 1
