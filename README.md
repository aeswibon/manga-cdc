# manga-cdc

Track manga releases from multiple scan sites and get notified when new chapters drop — via Discord, Slack, or Telegram.

## The Problem

Manga chapters are scattered across half a dozen scanlation sites (MangaDex, MangaFire, MangaPlus, Asura Scans, MangaPill, MangaTown). Each site has different update schedules, different APIs (or no API at all), and no unified way to track what's new. Manually checking each site daily is tedious and error-prone.

Manga-CDC solves this by acting as a **Change Data Capture pipeline for manga releases**: it scrapes all sources on a cron schedule, detects new chapters via a diff engine, and pushes notifications through your preferred channels — all in real-time via Kafka streaming.

## How It Works

```mermaid
sequenceDiagram
    participant Cron as Cron (every 30m)
    participant Adapters as Source Adapters (6 sites)
    participant Diff as Diff Engine
    participant DB as PostgreSQL
    participant Kafka as Kafka (Aiven)
    participant Notifier as Notification Service
    participant Channels as Discord / Slack / Telegram

    Cron->>Adapters: trigger scrape
    Adapters->>Adapters: fetch latest series + chapters
    Adapters->>Diff: raw series/chapter data
    Diff->>DB: query known state
    DB-->>Diff: existing series/chapters
    Diff->>Diff: compare → find new chapters
    alt new chapter found
        Diff->>DB: upsert series, insert chapter
        Diff->>Kafka: publish chapter event (Debezium format)
        Kafka-->>Notifier: deliver event
        Notifier->>Channels: send notification
        Notifier->>DB: mark chapter notified
    else no changes
        Diff->>Diff: skip
    end
```

## Architecture

```mermaid
flowchart TB
    subgraph Scraping["Scraper (Go)"]
        A1[MangaDex] --> E[Diff Engine]
        A2[MangaFire] --> E
        A3[MangaPlus] --> E
        A4[Asura Scans] --> E
        A5[MangaPill] --> E
        A6[MangaTown] --> E
        E --> DB[(Aiven PostgreSQL)]
    end

    subgraph Eventing["Eventing Layer"]
        DB --> K[Aiven Kafka]
    end

    subgraph Notifications["Notification Service (Spring Boot / Java)"]
        K --> NC[Kafka Consumer]
        NC --> NS[Notifier Router]
        NS --> DC[Discord]
        NS --> SL[Slack]
        NS --> TG[Telegram]
    end

    style Scraping fill:#1a73e8,color:#fff
    style Eventing fill:#ea4335,color:#fff
    style Notifications fill:#34a853,color:#fff
```

**Production deployment** uses [Aiven](https://aiven.io) for managed PostgreSQL and Kafka (SCRAM-SHA-256 over SASL_SSL).

For local development, two eventing backends are available (configured via the [setup wizard](#quick-start)):

| Backend | How it works |
|---------|-------------|
| **Kafka** | Scraper publishes Debezium-compatible JSON → Redpanda → notification service consumer |
| **QStash** | Scraper publishes via Upstash QStash HTTP API → Caddy → notification service webhook |

## Why This Stack

| Question | Answer |
|----------|--------|
| **Why Go for the scraper?** | Fast startup, low memory, excellent concurrency for parallel scraping, single binary deploy |
| **Why Spring Boot / Java for notifications?** | Rich ecosystem for notification integrations, JDBC/R2DBC, battle-tested Kafka client |
| **Why Kafka?** | Reliable at-least-once delivery, persistent event log, consumer group rebalancing, Debezium-compatible schema |
| **Why Aiven?** | Managed Kafka + Postgres under one provider, SCRAM-SHA-256 auth, no operational overhead |
| **Why both Kafka and QStash paths?** | Kafka for production-grade streaming; QStash for simpler local dev without a persistent broker |

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Scraper | Go 1.26, pgx, Colly, segmentio/kafka-go |
| Database | Aiven PostgreSQL 16 |
| Eventing | Aiven Kafka (SASL_SSL / SCRAM-SHA-256) |
| Notifications | Spring Boot 3.3, Java 21 |
| Notifier targets | Discord, Slack, Telegram |
| Metrics | Prometheus + Grafana |
| Deployment | Docker Compose, Kubernetes/Helm, Terraform/GCP |
| Orchestration | GitHub Actions CI/CD |

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
├── configure/                  # Setup wizard (Go CLI)
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
│       └── repository/         # JDBC data access
├── connectors/                 # Debezium connector configs
├── db/migrations/              # SQL schema migrations
├── helm/                       # Kubernetes Helm chart
├── terraform/                  # GCP Terraform IaC
├── docker-compose.yml          # Local dev compose (generated)
├── docker-compose.prod.yml     # Production compose (generated)
└── prometheus.yml              # Metrics scraping config
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

## Production (Aiven)

The production deployment uses [Aiven](https://aiven.io) for both PostgreSQL and Kafka:

- **Aiven PostgreSQL** — scraper connects via `DATABASE_URL` (postgres:// with SSL); notification service connects via derived JDBC URL
- **Aiven Kafka** — scraper publishes chapter events using SCRAM-SHA-256 over SASL_SSL; notification service consumes from the same topic

**Required secrets** in GitHub Actions:
- `DATABASE_URL`, `KAFKA_BROKERS`, `KAFKA_USERNAME`, `KAFKA_PASSWORD`
- Discord/Slack/Telegram webhook tokens

## Local Development

For local development, two eventing backends are available via the [setup wizard](#quick-start):

### Kafka Mode (local)

- Scraper publishes to self-hosted Redpanda in Docker Compose
- Notification service consumes from Redpanda
- Requires: Redpanda, Kafka Connect, Debezium PostgreSQL connector

### QStash Mode (local)

- Scraper publishes via Upstash QStash HTTP API
- QStash delivers to Caddy → notification service webhook
- Requires: Caddy, QStash account (free tier available)

## License

MIT
