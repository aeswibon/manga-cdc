#!/usr/bin/env bash
set -euo pipefail

# Full stack end-to-end test for manga-cdc
# Prerequisites: Docker, Go, Java 21+, jq, psql
#
# Usage:
#   ./scripts/e2e.sh              # run full E2E test
#   ./scripts/e2e.sh --skip-build # skip Go/Java builds

SKIP_BUILD="${1:-}"

cd "$(dirname "$0")/.."
ROOT=$(pwd)
LOG="$ROOT/target/e2e.log"

mkdir -p target
echo "=== manga-cdc E2E Test ===" | tee "$LOG"

cleanup() {
  echo "Cleaning up..." | tee -a "$LOG"
  cd "$ROOT"
  docker compose down -v 2>/dev/null || true
}
trap cleanup EXIT

if [ "$SKIP_BUILD" != "--skip-build" ]; then
  echo "Building Go scraper..." | tee -a "$LOG"
  (cd scraper && go build -o ../target/scraper ./cmd/scraper/) 2>&1 | tee -a "$LOG"

  echo "Building notification service..." | tee -a "$LOG"
  (cd notification-service && mvn -q package -DskipTests) 2>&1 | tee -a "$LOG"
fi

echo "Starting services via Docker Compose..." | tee -a "$LOG"
docker compose up -d postgres redpanda connect 2>&1 | tee -a "$LOG"

echo "Waiting for services to be healthy..." | tee -a "$LOG"
for i in $(seq 1 30); do
  if docker compose exec postgres pg_isready -U mangacdc >/dev/null 2>&1; then
    echo "PostgreSQL ready" | tee -a "$LOG"
    break
  fi
  sleep 2
done

echo "Registering Debezium connector..." | tee -a "$LOG"
curl -s -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d @connectors/postgres-connector.json 2>&1 | tee -a "$LOG"

echo "Setting up test data..." | tee -a "$LOG"
docker compose exec -T postgres psql -U mangacdc -d mangacdc <<SQL
  INSERT INTO manga_series (id, source_id, title, source_url, status, is_active)
  VALUES ('00000000-0000-0000-0000-000000000001', 'e2e-source', 'E2E Test Series', 'https://example.com/e2e', 'ONGOING', true)
  ON CONFLICT (source_id) DO NOTHING;

  INSERT INTO chapters (id, series_id, chapter_num, title, url, is_new)
  VALUES ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001', 1, 'Chapter 1', 'https://example.com/e2e/ch-1', true)
  ON CONFLICT (series_id, chapter_num) DO NOTHING;
SQL

echo "Waiting for Debezium to capture change..." | tee -a "$LOG"
sleep 5

echo "Checking Kafka topics..." | tee -a "$LOG"
docker compose exec redpanda rpk topic list 2>&1 | tee -a "$LOG"

TOPIC="mangacdc.public.chapters"
MESSAGES=$(docker compose exec redpanda rpk topic consume "$TOPIC" --num 1 --timeout 5s 2>/dev/null || echo "")
if echo "$MESSAGES" | grep -q '"op":"c"'; then
  echo "PASS: CDC event captured in Kafka" | tee -a "$LOG"
else
  echo "FAIL: No CDC event found in Kafka" | tee -a "$LOG"
  exit 1
fi

echo "Starting notification service..." | tee -a "$LOG"
CDC_ENABLED=true \
SPRING_KAFKA_BOOTSTRAP_SERVERS=localhost:9092 \
SPRING_DATASOURCE_URL=jdbc:postgresql://localhost:5432/mangacdc \
DISCORD_WEBHOOK_URL="http://localhost:9999/mock" \
java -jar notification-service/target/notification-service-0.0.1-SNAPSHOT.jar &
NOTIFY_PID=$!
sleep 10

echo "Checking notification logs..." | tee -a "$LOG"
STATUS=$(docker compose exec -T postgres psql -U mangacdc -d mangacdc -tAc \
  "SELECT status FROM notification_logs WHERE channel='discord' AND chapter_id='00000000-0000-0000-0000-000000000101'")

if [ "$STATUS" = "SENT" ] || [ "$STATUS" = "FAILED" ]; then
  echo "PASS: Notification logged (status=$STATUS)" | tee -a "$LOG"
else
  echo "FAIL: No notification log found" | tee -a "$LOG"
  exit 1
fi

kill "$NOTIFY_PID" 2>/dev/null || true
wait "$NOTIFY_PID" 2>/dev/null || true

echo ""
echo "=== E2E Test Complete ===" | tee -a "$LOG"
