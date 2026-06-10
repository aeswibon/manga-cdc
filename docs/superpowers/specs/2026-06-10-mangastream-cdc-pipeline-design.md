# MangaStream CDC Pipeline — Design Spec

## Overview

A production-grade manga chapter notification system built with Change Data Capture (CDC) architecture. Phase 1 delivers an end-to-end working system (scraper → PostgreSQL → Discord webhook) with direct DB polling. Phase 2 introduces the full CDC pipeline (Debezium → Kafka).

## Goals

- **Real** — Actually tracks manga releases and sends notifications
- **Impressive** — Production-grade architecture: containerized, observable, resilient
- **Educational** — Demonstrates CDC, event-driven architecture, distributed systems patterns

## Phase 1 Architecture

```
[Go Scraper] → [PostgreSQL] → [Spring Boot Notification Service] → [Discord Webhook]
                   ↓
          [Prometheus + Grafana]
```

## Phase 2 Architecture (Preview)

```
[Go Scraper] → [PostgreSQL] → [WAL] → [Debezium] → [Kafka] → [Notification Service] → [Discord]
```

## Deployment

- **Phase 1:** Docker Compose (local dev)
- **Phase 2:** Kubernetes manifests + Helm chart (production target)

## Data Model

### manga_series
| Column | Type | Notes |
|--------|------|-------|
| id | UUID PRIMARY KEY | |
| source_id | VARCHAR(255) UNIQUE NOT NULL | Source site's ID |
| title | VARCHAR(500) NOT NULL | |
| alt_titles | JSONB | Alternative titles/aliases |
| author | VARCHAR(255) | |
| artist | VARCHAR(255) | |
| description | TEXT | |
| cover_url | TEXT | |
| status | VARCHAR(20) | ONGOING, COMPLETED, HIATUS, CANCELLED |
| source_url | TEXT NOT NULL | URL on the source site |
| latest_chapter | DECIMAL(10,1) | |
| last_checked | TIMESTAMPTZ | When scraper last polled |
| is_active | BOOLEAN DEFAULT true | |
| created_at | TIMESTAMPTZ DEFAULT NOW() | |
| updated_at | TIMESTAMPTZ DEFAULT NOW() | |

### chapters
| Column | Type | Notes |
|--------|------|-------|
| id | UUID PRIMARY KEY | |
| series_id | UUID FK → manga_series(id) | |
| chapter_num | DECIMAL(10,1) NOT NULL | Supports .5 chapters |
| title | VARCHAR(500) | |
| url | TEXT NOT NULL | URL to read this chapter |
| release_date | TIMESTAMPTZ | |
| is_new | BOOLEAN DEFAULT true | Flagged until notified |
| created_at | TIMESTAMPTZ DEFAULT NOW() | |
| UNIQUE(series_id, chapter_num) | | Prevents duplicates |

### notification_log
| Column | Type | Notes |
|--------|------|-------|
| id | UUID PRIMARY KEY | |
| chapter_id | UUID FK → chapters(id) | |
| status | VARCHAR(20) | PENDING, SENT, FAILED |
| channel | VARCHAR(50) | discord, telegram, slack |
| error_message | TEXT | |
| sent_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ DEFAULT NOW() | |

## Scraper Service (Go)

### Architecture
- **Source Adapters:** Interface per manga source (MangaDex, MangaPlus, custom)
- **Diff Engine:** Compares scraped data vs DB state, detects new chapters
- **DB Writer:** Upserts series + batch inserts chapters via pgx
- **Config:** YAML/env config for source URLs, intervals, credentials

### Scraper Loop
1. **Fetch** — HTTP GET source's latest updates endpoint
2. **Parse** — Extract data via source adapter
3. **Diff** — Compare against DB (query latest known chapter per series)
4. **Insert** — Write new chapters, upsert series metadata
5. **Sleep** — Configurable interval (e.g., 5 min)

### Error Handling
- Transient errors: exponential backoff with jitter
- Persistent errors: Prometheus metric alert, graceful degradation
- DB failures: pgx connection pool with health check, startup blocks on DB readiness
- Duplicates: UNIQUE constraint + INSERT ON CONFLICT DO NOTHING

## Notification Service (Spring Boot)

- REST API for querying new chapters
- Discord webhook integration (rich embeds with cover art)
- Direct DB polling as interim notification mechanism (Phase 1)
- Will be replaced by Kafka consumer in Phase 2

## Implementation Roadmap

### Phase 1.1 — Scaffold & Schema (Day 1)
- Initialize Go module and Spring Boot project
- PostgreSQL migrations
- Docker Compose with PostgreSQL + pgAdmin

### Phase 1.2 — First Source Adapter (Day 2-3)
- MangaDex adapter
- Diff engine + DB writer
- Scraper running in Docker, populating PostgreSQL

### Phase 1.3 — Second Source & Observability (Day 4-5)
- MangaPlus adapter (proves adapter pattern)
- Prometheus metrics endpoint
- Grafana dashboard

### Phase 1.4 — Notification Service (Day 6-7)
- Spring Boot service with Discord webhook
- Direct DB polling for notifications
- End-to-end: scraper → DB → Discord

## Dev Environment

```yaml
# docker-compose.yml (Phase 1)
services:
  postgres:
    image: postgres:16-alpine
    ports: ["5432:5432"]
    volumes: [pgdata, ./db/migrations:/docker-entrypoint-initdb.d]

  scraper:
    build: ./scraper
    depends_on: [postgres]
    environment: [DATABASE_URL, SCRAPE_INTERVAL]

  notification-service:
    build: ./notification-service
    depends_on: [postgres]
    ports: ["8080:8080"]

  prometheus:
    image: prom/prometheus
    ports: ["9090:9090"]

  grafana:
    image: grafana/grafana
    ports: ["3000:3000"]

volumes:
  pgdata:
```

## Project Structure

```
manga-cdc/
├── scraper/                  # Go module
│   ├── cmd/scraper/          # main.go
│   ├── internal/
│   │   ├── adapter/          # Source adapters
│   │   ├── model/            # Domain types
│   │   ├── db/               # pgx queries
│   │   ├── diff/             # Change detection
│   │   └── config/           # Config loader
│   ├── Dockerfile
│   └── go.mod
├── notification-service/     # Spring Boot
├── db/
│   └── migrations/           # SQL migration files
└── docker-compose.yml
```

## Testing Strategy
- **Unit tests (Go):** Adapter parsing, diff engine, DB queries. Mock HTTP and DB.
- **Integration tests:** Testcontainers — real PostgreSQL, recorded API responses.
- **E2E test:** docker-compose up, verify scraper cycle, confirm webhook fires.

## Phase 2 (Future)
- Replace direct DB polling with Debezium → Kafka pipeline
- Notification service becomes Kafka consumer
- Add Kubernetes manifests + Helm chart
- Exactly-once notification semantics via CDC offset tracking
