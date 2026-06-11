# manga-cdc

Track manga releases from multiple sources and get notified when new chapters drop — via Discord, Slack, or Telegram.

## Architecture

```
                          ┌──────────────────┐
                          │   Source Adapters │
                          │ (MangaDex, Fire,  │
                          │  Plus, Asura,     │
                          │  Town, Pill)      │
                          └────────┬─────────┘
                                   │
                                   ▼
                          ┌──────────────────┐
                          │  Go Scraper       │
                          │  (diff engine)    │
                          └────────┬─────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    ▼              ▼              ▼
           ┌────────────┐  ┌───────────┐  ┌──────────┐
           │ PostgreSQL │  │  Kafka    │  │  QStash  │
           │ (canonical │  │ (optional)│  │(optional)│
           │  store)    │  └─────┬─────┘  └────┬─────┘
           └─────┬──────┘        │             │
                 │               ▼             ▼
                 │        ┌──────────┐  ┌──────────┐
                 │        │ Redpanda │  │  Caddy   │
                 │        └────┬─────┘  └────┬─────┘
                 │             │             │
                 ▼             ▼             ▼
          ┌──────────────────────────────────────┐
          │      Notification Service            │
          │  (Spring Boot — Kafka Consumer       │
          │   + Webhook Receiver)               │
          └──────────────┬───────────────────────┘
                         │
                         ▼
          ┌──────────┐ ┌───────┐ ┌─────────┐
          │ Discord  │ │ Slack │ │ Telegram│
          └──────────┘ └───────┘ └─────────┘

                    ┌──────────────────────┐
                    │ Prometheus + Grafana │
                    │   (observability)    │
                    └──────────────────────┘
```

**Two eventing backends supported:**

| Backend | How it works |
|---------|-------------|
| **Kafka** | Scraper publishes Debezium-compatible JSON → Redpanda → notification service consumer → webhook |
| **QStash** | Scraper publishes via Upstash QStash HTTP API → Caddy reverse proxy → notification service webhook endpoint |

Use the [setup wizard](#quick-start) to choose your configuration.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Scraper | Go 1.23, pgx, Colly |
| Database | PostgreSQL 16 |
| Eventing (optional) | Redpanda/Kafka or Upstash QStash + Caddy |
| Notifications | Spring Boot 3.3, Java 21 |
| Notifier targets | Discord, Slack, Telegram |
| Metrics | Prometheus + Grafana |
| Deployment | Docker Compose, Kubernetes/Helm, Terraform/GCP |

## Quick Start

```bash
# Clone the repo
git clone https://github.com/aeswibon/manga-cdc.git
cd manga-cdc

# Run the setup wizard
go run ./configure

# Follow the generated guide
cat SETUP.md
```

## Project Structure

```
manga-cdc/
├── configure/                  # ✨ Setup wizard (Go CLI)
├── scraper/                    # Go scraper module
│   ├── cmd/scraper/            # Scraper entrypoint
│   ├── internal/
│   │   ├── adapter/            # Source adapters (6 sources)
│   │   ├── model/              # Domain types
│   │   ├── db/                 # PostgreSQL client (pgx)
│   │   ├── diff/               # Change detection engine
│   │   ├── kafka/              # Kafka producer (optional)
│   │   ├── qstash/             # QStash publisher (optional)
│   │   └── config/             # Env-based config
├── notification-service/       # Spring Boot notification service
│   └── src/main/java/com/mangacdc/
│       ├── controller/         # Webhook endpoint for QStash
│       ├── service/            # Kafka consumer + notifiers
│       ├── repository/         # JDBC data access
│       └── config/             # Kafka consumer config
├── connectors/                 # Debezium connector configs
├── db/migrations/              # SQL schema migrations
├── helm/                       # Kubernetes Helm chart
├── terraform/                  # GCP Terraform IaC
├── docker-compose.yml          # Local dev compose (generated)
├── docker-compose.prod.yml     # Production compose (generated)
├── prometheus.yml              # Metrics scraping config
└── docs/superpowers/
    ├── specs/                  # Design documents
    └── plans/                  # Implementation plans
```

## Development

### Without the wizard

```bash
# Start PostgreSQL
docker compose up -d postgres

# Run scraper (Go)
cd scraper && go run ./cmd/scraper

# Run notification service (Java)
cd notification-service && ./mvnw spring-boot:run
```

### Environment Variables

See `.env.example` (generated by the setup wizard) for all available options.

### Adding a New Source

Implement the `SourceAdapter` interface in `scraper/internal/adapter/`:

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

## Eventing Backends

### Kafka Mode

- Scraper publishes chapter events as Debezium-compatible JSON to Redpanda/Kafka
- Notification service consumes from a Kafka topic via `@KafkaListener`
- Requires: Redpanda, Kafka Connect, Debezium PostgreSQL connector

### QStash Mode

- Scraper publishes chapter events via Upstash QStash HTTP API
- QStash delivers to the configured webhook URL via Caddy reverse proxy
- Notification service receives via `POST /api/webhook`
- Requires: Caddy, QStash account (free tier available)

### No Eventing (DB Polling)

- Notification service polls `chapters WHERE is_new = true` directly
- No external eventing dependencies required
- Simpler but higher latency

## License

MIT
