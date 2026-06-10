#!/bin/bash
set -euo pipefail

# Registers the Debezium PostgreSQL CDC connector with Redpanda.
# Usage: ./register-connector.sh

CONNECT_URL="${CONNECT_URL:-http://localhost:8083}"
CONNECTOR_NAME="${CONNECTOR_NAME:-mangacdc-connector}"
COMPOSE_SERVICE="${COMPOSE_SERVICE:-notification-service}"

echo "Waiting for Debezium connect API at $CONNECT_URL..."
until curl -sf "$CONNECT_URL/connectors" > /dev/null 2>&1; do
  sleep 2
done
echo "Connect API ready."

echo "Registering connector '$CONNECTOR_NAME'..."
curl -sf -X POST "$CONNECT_URL/connectors" \
  -H "Content-Type: application/json" \
  -d @connectors/debezium-postgres.json > /dev/null
echo "Connector registered."

echo "Waiting for connector to reach RUNNING state..."
until [ "$(curl -sf "$CONNECT_URL/connectors/$CONNECTOR_NAME/status" | python3 -c "import sys,json; print(json.load(sys.stdin)['connector']['state'])")" = "RUNNING" ]; do
  sleep 2
done
echo "Connector is RUNNING."

echo "Waiting for CDC topics to be created..."
for topic in "mangacdc.public.chapters" "mangacdc.public.manga_series" "mangacdc.public.notification_logs"; do
  until docker compose exec redpanda rpk topic list 2>/dev/null | grep -q "$topic"; do
    sleep 2
  done
  echo "  Topic '$topic' created."
done

echo "Restarting $COMPOSE_SERVICE to pick up CDC topics..."
docker compose restart "$COMPOSE_SERVICE" > /dev/null 2>&1
echo "Done. CDC pipeline is ready."
