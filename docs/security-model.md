# Security model

This project is **public on GitHub**. Anyone can read the architecture, API shapes, and deployment patterns. Security therefore depends on **secrets, configuration, and trust boundaries** — not on hiding how the code works.

## What the public repo reveals

| Public knowledge | Risk if misconfigured |
|------------------|------------------------|
| Debezium-style webhook JSON (`op`, `after.id`, `after.url`, `is_new`) | Forged notifications if webhook auth is off |
| Read API paths (`/api/series`, `/api/logs`, `/api/stats`, SSE stream) | Data harvesting if `SECURITY_REQUIRE_API_KEY=false` |
| Service naming (`manga-cdc-notifier-<env>`, Vercel app names in docs) | Easier targeting / probing |
| Kafka topic `mangacdc.public.chapters` | Fake events if Kafka credentials leak |
| Local defaults `mangacdc:mangacdc` | Spray attacks on accidentally exposed dev DB |

## What is NOT in the repo (must stay secret)

- `API_READ_KEY`, `WEBHOOK_SECRET`, QStash signing keys, `ADMIN_API_KEY`
- `DATABASE_URL`, Kafka passwords, Discord/Slack/Telegram credentials
- Vercel `NOTIFIER_URL`, `NOTIFIER_API_KEY`, `PIPELINE_HEALTH_URL`

## Trust boundaries

### 1. Notification service (Cloud Run / VM)

Production defaults (Terraform GCP, `docker-compose.prod.yml`):

- `SECURITY_REQUIRE_API_KEY=true` — all `/api/*` reads need `X-Api-Key`
- `SECURITY_REQUIRE_WEBHOOK_AUTH=true` — webhook needs QStash signature or `WEBHOOK_SECRET`
- `ADMIN_MUTATIONS_ENABLED=false` — no series/log mutations without explicit enable + `ADMIN_API_KEY`
- Rate limits on read, webhook, and SSE connections
- Webhook payloads must match a real `is_new` chapter **and** the stored chapter URL in Postgres

Public without app auth:

- `GET /actuator/health` only (liveness)

### 2. Dashboard (Vercel)

The dashboard proxy (`/api/notifier/*`) injects `NOTIFIER_API_KEY` server-side. It is **not** a secret vault:

- Anyone can call the same read endpoints the UI uses (GET-only allowlist)
- Mutations, webhooks, and admin paths are blocked at the proxy
- Client `X-Api-Key` / `X-Admin-Key` headers are never forwarded

Treat dashboard data as **intentionally public read** if the dashboard URL is public.

### 3. Status page (Vercel)

- `GET /api/status` is public by design
- Backend `PIPELINE_HEALTH_URL` and `NOTIFIER_API_KEY` stay server-side
- Health details are sanitized (no raw DB/Kafka exception text)

### 4. Scraper

- No HTTP trigger for scrapes
- `WATCHLIST_URL` (if used) is HTTPS + host allowlist only

## Operator checklist

1. Set GitHub secrets: `API_READ_KEY`, `WEBHOOK_SECRET`, `QSTASH_CURRENT_SIGNING_KEY`, `QSTASH_NEXT_SIGNING_KEY`
2. Set Vercel env: `NOTIFIER_URL`, `NOTIFIER_API_KEY`, `PIPELINE_HEALTH_URL` (status page)
3. Set repo variable `ALLOWED_ORIGINS` to your dashboard/status origins (comma-separated)
4. Never deploy with `SECURITY_REQUIRE_API_KEY=false` or `SECURITY_REQUIRE_WEBHOOK_AUTH=false` in production
5. Never enable `ADMIN_MUTATIONS_ENABLED` in prod unless you also set a strong `ADMIN_API_KEY`
6. Rotate keys if you suspect exposure; updating GitHub/Vercel secrets is enough — nothing live belongs in git

## Residual risks (accepted or follow-up)

- **Cloud Run `allUsers` invoker** — required for QStash/public health today; app-layer auth must stay on
- **Public dashboard proxy** — exposes the same read data as the UI; acceptable for a public watchlist dashboard
- **URL enumeration** — docs use placeholders; live hostnames still discoverable via Vercel/DNS
- **Deploy workflow_dispatch** — restrict via GitHub Environment protection on `production`
