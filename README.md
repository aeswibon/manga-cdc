# manga-cdc

A production-grade manga chapter notification system built with Change Data Capture (CDC) architecture.

Track manga releases from multiple sources and get notified via Discord when new chapters drop — powered by Go, PostgreSQL, Debezium, and Redpanda/Kafka.

## Architecture

```
[Go Scraper] → [PostgreSQL] → [WAL] → [Debezium] → [Redpanda] → [Spring Boot Consumer] → [Discord Webhook]
       │                        │
       ▼                        ▼
[Prometheus + Grafana]    [notification_logs]
```

- **Scraper** (Go) — Polls MangaDex API, diffs against DB state, inserts new chapters
- **PostgreSQL** — Central data store; Write-Ahead Log (WAL) feeds CDC
- **Debezium** — Streams WAL changes to Redpanda via Kafka Connect
- **Redpanda** — Kafka-compatible event backbone
- **Notification Service** (Spring Boot / Java 21) — Consumes CDC events, dispatches Discord embeds
- **Observability** — Prometheus metrics + Grafana dashboards

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Scraper | Go 1.23, pgx, Colly |
| Database | PostgreSQL 16 |
| CDC | Debezium 2.7, Kafka Connect |
| Event Stream | Redpanda 24.2 |
| Notifications | Spring Boot 3.3, Java 21 |
| Metrics | Prometheus + Grafana |
| Containerization | Docker Compose |
| Deployment (future) | Kubernetes + Terraform |

## Getting Started

### Prerequisites

- Docker & Docker Compose
- Git

### Clone & Run

```bash
git clone https://github.com/aeswibon/manga-cdc.git
cd manga-cdc
docker compose up --build -d
```

This starts all 8 services: postgres, redpanda, connect (Debezium), scraper, notification-service, kafkaui, prometheus, grafana.

### First Scrape

The scraper runs immediately on startup and then every 5 minutes. Check the logs:

```bash
docker compose logs scraper
# Expected: "scraper started" with MangaDex source, "new chapters detected"
```

### Enable Discord Notifications

```bash
DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/your-webhook-id/your-webhook-token" \
  docker compose up -d notification-service
```

### Verify CDC Pipeline

```bash
# Check Debezium connector status
curl -s http://localhost:8083/connectors/mangacdc-connector/status | jq

# Consume a CDC message
docker compose exec redpanda rpk topic consume mangacdc.public.chapters --num 1

# Check notification logs
docker compose exec postgres psql -U mangacdc -d mangacdc \
  -c "SELECT COUNT(*) FROM notification_logs WHERE status = 'SENT';"
```

## Project Structure

```
manga-cdc/
├── scraper/                    # Go scraper module
│   ├── cmd/scraper/main.go     # Entrypoint
│   └── internal/
│       ├── adapter/            # Source adapters (MangaDex, extensible)
│       ├── model/              # Domain types
│       ├── db/                 # PostgreSQL client (pgx)
│       ├── diff/               # Change detection engine
│       └── config/             # Env-based config
├── notification-service/       # Spring Boot notification service
│   └── src/main/java/com/mangacdc/
│       ├── config/             # Kafka consumer config
│       ├── model/              # Chapter data record
│       ├── repository/         # JDBC data access
│       └── service/            # Discord notifier + Kafka consumer
├── connectors/                 # Debezium connector configs
├── db/migrations/              # SQL schema migrations
├── docker-compose.yml          # All services
├── prometheus.yml              # Metrics scraping config
└── docs/
    └── superpowers/
        ├── specs/              # Design documents
        └── plans/              # Implementation plans
```

## Development

### Local Development (without Docker)

```bash
# Terminal 1: PostgreSQL
docker compose up -d postgres

# Terminal 2: Scraper
cd scraper && go run ./cmd/scraper

# Terminal 3: Notification Service (with Discord webhook)
cd notification-service
./mvnw spring-boot:run -Dspring-boot.run.arguments=--discord.webhook-url=$WEBHOOK_URL
```

### Adding a New Source

Implement the `SourceAdapter` interface:

```go
type SourceAdapter interface {
    Name() string
    FetchLatest(ctx context.Context) ([]model.Series, error)
    FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error)
}
```

## Dashboard Access

| Service | URL |
|---------|-----|
| Kafka UI | http://localhost:8085 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |

## Phase 2: CDC Pipeline

The DB polling mechanism (Phase 1) was replaced with a CDC pipeline (Phase 2):

- **Before:** Notification service polled `chapters WHERE is_new = true` every 30s
- **After:** Debezium streams WAL changes → Redpanda → Kafka consumer triggers notifications in real-time

## Contributing

1. Fork the repo
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes (`git commit -m "feat: add amazing feature"`)
4. Push (`git push origin feat/amazing-feature`)
5. Open a Pull Request

## License

MIT
