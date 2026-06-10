# manga-cdc

A production-grade manga chapter notification system built with Change Data Capture (CDC) architecture.

Track manga releases from multiple sources and get notified via Discord when new chapters drop — powered by PostgreSQL, Debezium, and Kafka.

## Architecture

```
Scraper → PostgreSQL → [WAL] → Debezium → Kafka → Notification Service → Discord
```

## Getting Started

```bash
docker compose up --build
```
