# QStash Event Architecture

**Status:** Implemented (retrospective spec)

**Motivation:** Replace Upstash Kafka (managed Kafka) with a simpler HTTP-based eventing model. Kafka required persistent TCP connections, SASL/SCRAM auth, and a heavier notification consumer. QStash is HTTP-only — the scraper POSTs events, QStash retries on failure, and the notification service receives via a standard webhook endpoint.

---

## Architecture

```
┌──────────┐     POST /v1/publish/     ┌────────────┐
│  Scraper │ ──── Authorization ──────→ │   QStash   │
│ (Go)     │     Upstash-Destination    │ (Upstash)  │
└──────────┘     Content-Type: json     └──────┬─────┘
                                               │ delivery (retries on failure)
                                               ▼
                                      ┌────────────────┐
                                      │     Caddy      │
                                      │ reverse proxy  │
                                      └──────┬─────────┘
                                             │
                                             ▼
                              ┌──────────────────────────┐
                              │  Notification Service    │
                              │  POST /api/webhook        │
                              │  WebhookController        │
                              │  → ChapterEventService    │
                              │  → NotifierRegistry       │
                              │  → Discord / Slack / Tel  │
                              └──────────────────────────┘
```

## Components

### 1. QStash Publisher (`scraper/internal/qstash/publisher.go`)

Publishes Debezium-compatible chapter events to Upstash QStash via HTTP POST.

**Interface:**

```go
type Publisher struct {
    client      HTTPClient     // pluggable for tests
    token       string         // QSTASH_TOKEN
    destination string         // QSTASH_DESTINATION_URL
    apiURL      string         // https://qstash.upstash.io/v1/publish/
}

func NewPublisher(token, destination string) *Publisher
func (p *Publisher) PublishChapterEvent(ctx context.Context, chapter model.Chapter) error
```

**Payload format (Debezium-compatible JSON):**

```json
{
    "op": "c",
    "after": {
        "id": "uuid",
        "series_id": "uuid",
        "chapter_num": 1.0,
        "title": "...",
        "url": "https://...",
        "is_new": true
    }
}
```

**Headers sent:**
- `Authorization: Bearer <token>` — QStash API token
- `Upstash-Destination: <url>` — where QStash delivers the payload
- `Content-Type: application/json`

**Wiring** (`scraper/cmd/scraper/main.go:95-99`):
```go
var qstashPublisher *qstash.Publisher
if cfg.QStashToken != "" && cfg.QStashDestination != "" {
    qstashPublisher = qstash.NewPublisher(cfg.QStashToken, cfg.QStashDestination)
}
```

Published inside the chapter loop alongside the optional Kafka producer — both can be active simultaneously.

### 2. Caddy Reverse Proxy (`Caddyfile`)

Listens on ports 80/443 and proxies QStash deliveries to the notification service:

```
mangacdc.34.73.231.106.sslip.io {
    reverse_proxy notification-service:8080
}
```

The notification service runs on port 8080 internally. Caddy provides TLS termination and public exposure via `sslip.io` (free DNS pointing to the GCP VM's ephemeral IP).

### 3. Webhook Controller (`notification-service/.../WebhookController.java`)

Receives QStash deliveries at `POST /api/webhook`:

```java
@PostMapping("/webhook")
public ResponseEntity<String> handleWebhook(@RequestBody String message) {
    chapterEventService.processChapterEvent(message);
    return ResponseEntity.ok("OK");
}
```

### 4. Chapter Event Service (`notification-service/.../ChapterEventService.java`)

Shared processing logic used by both the webhook endpoint and the Kafka consumer:

1. Parses JSON, validates `op == "c"` and `is_new == true`
2. Looks up series title from PostgreSQL via JDBC
3. Dispatches to all configured notifiers via `NotifierRegistry`
4. Logs notification result to `notification_log` table
5. Marks chapter as notified (`is_new = false`)

### 5. Notifier Registry (`notification-service/.../NotifierRegistry.java`)

Iterates all registered `Notifier` implementations and calls `sendChapterAlert()` on each that is configured. Currently supports:
- Discord (`DiscordNotifier`)
- Slack (`SlackNotifier`)
- Telegram (`TelegramNotifier`)

---

## Configuration

### Environment Variables (Production)

| Variable | Source | Required |
|---|---|---|
| `QSTASH_TOKEN` | Upstash dashboard | Yes |
| `QSTASH_DESTINATION_URL` | Your exposed webhook URL | Yes |
| `DISCORD_WEBHOOK_URL` | Discord channel settings | For Discord |
| `SLACK_WEBHOOK_URL` | Slack workspace settings | For Slack |
| `TELEGRAM_BOT_TOKEN` | @BotFather | For Telegram |
| `TELEGRAM_CHAT_ID` | Your chat with the bot | For Telegram |

### Docker Compose (Production)

`docker-compose.prod.yml` runs four services: postgres, caddy, scraper, notification-service.
QStash env vars are passed to the scraper service.

### CI/CD

`.github/workflows/gcloud-deploy.yml` generates a `.env` file on the GCP VM containing all QStash, Discord, Slack, Telegram secrets from GitHub Actions secrets, then runs `docker compose -f docker-compose.prod.yml up -d`.

---

## Reliability

- **QStash retries:** Upstash QStash automatically retries deliveries on HTTP 5xx or network errors. The webhook endpoint returns 200 OK on success.
- **Scraper no-ops if QStash is down:** The scraper logs errors but continues scraping and writing to PostgreSQL. Chapters are not lost — they remain `is_new = true` in the DB until notified.
- **Caddy auto-TLS:** Caddy automatically provisions Let's Encrypt certificates for the public domain.
- **No at-least-once guarantee:** In practice QStash delivers at-least-once. The notification service is idempotent (checks `is_new` before notifying).

---

## Comparison with Kafka Path

| Aspect | QStash | Kafka |
|---|---|---|
| Transport | HTTP POST | Persistent TCP |
| Auth | Bearer token | SASL/SCRAM-SHA-256 |
| Scalability | Upstash-managed | Self-managed Redpanda |
| Complexity | Simple (3 components) | Complex (Redpanda + Connect + Debezium) |
| Retry | Built-in (QStash) | Consumer group rebalance |
| Cost | Free tier available | Self-hosted infra cost |
| Latency | ~100ms (HTTP round trip) | ~10ms (persistent conn) |

---

## File Reference

| File | Role |
|---|---|
| `scraper/internal/qstash/publisher.go` | QStash HTTP publisher |
| `scraper/internal/qstash/publisher_test.go` | Unit tests (3 tests) |
| `scraper/cmd/scraper/main.go:95-99` | Conditional wiring |
| `scraper/internal/config/config.go:57-58` | QStash config fields |
| `Caddyfile` | Reverse proxy config |
| `docker-compose.prod.yml` | Production services |
| `notification-service/.../WebhookController.java` | Webhook endpoint |
| `notification-service/.../ChapterEventService.java` | Shared event processing |
| `.github/workflows/gcloud-deploy.yml` | QStash secrets in CD |
