#!/bin/bash
set -euo pipefail

# One-command setup: build, start, register connector, restart notification service.
# Usage: ./setup.sh

echo "=== Building Docker images ==="
docker compose build

echo "=== Starting all services ==="
docker compose up -d

echo "=== Registering Debezium connector ==="
./register-connector.sh

echo "=== All services ready ==="
echo "  Scraper metrics:  http://localhost:2112/metrics"
echo "  Kafka UI:         http://localhost:8085"
echo "  Prometheus:       http://localhost:9090"
echo "  Grafana:          http://localhost:3000"
echo "  Debezium API:     http://localhost:8083"
echo "  Notification API: http://localhost:8080"
