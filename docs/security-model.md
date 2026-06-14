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
- Vercel `NOTIFIER_URL`, `NOTIFIER_API_KEY`
- GitHub `PIPELINE_HEALTH_URL`, `API_READ_KEY`, `KV_REST_API_URL`, `KV_REST_API_TOKEN` (health poller → KV sync)

## Trust boundaries

### 1. Notification service (Cloud Run / VM)

Production defaults (Terraform GCP, `docker-compose.prod.yml`):

- `SECURITY_REQUIRE_API_KEY=true` — all `/api/*` reads need `X-Api-Key`
- `SECURITY_REQUIRE_WEBHOOK_AUTH=true` — webhook needs QStash signature or `WEBHOOK_SECRET`
- `ADMIN_MUTATIONS_ENABLED=false` — no series/log mutations without explicit enable + `ADMIN_API_KEY`
- Rate limits on read, webhook, and SSE connections (client IP from `X-Forwarded-For` when present)
- Webhook payloads must match a real `is_new` chapter **and** the stored chapter URL in Postgres

Public without app auth:

- `GET /actuator/health` only (liveness)

### 2. Dashboard (Vercel)

The dashboard proxy (`/api/data/*`) injects `NOTIFIER_API_KEY` server-side. It is **not** a secret vault:

- Anyone can call the same read endpoints the UI uses (GET-only allowlist)
- Mutations, webhooks, and admin paths are blocked at the proxy
- Client `X-Api-Key` / `X-Admin-Key` headers are never forwarded
- Cover proxy (`/api/cover`) only fetches allowlisted CDN hosts (MangaDex, MangaPlus) with validated redirects

Treat dashboard data as **intentionally public read** if the dashboard URL is public.

### 3. Status page (Vercel)

- `GET /api/status` is public by design
- Production health is **not** fetched from the browser; GitHub Actions polls `/api/pipeline/health` with `X-Api-Key` every 5 minutes and writes sanitized JSON to **Vercel KV**
- The status page Edge function reads KV only — notifier URL and API key never ship to the client
- Health details are sanitized (no raw DB/Kafka exception text)

### 4. Scraper

- No HTTP trigger for scrapes
- `WATCHLIST_URL` (if used) is HTTPS + host allowlist; DNS resolution failures reject the URL
- `FLARESOLVERR_URL` must point to internal/loopback addresses only (Docker service name or localhost)
- CBZ archiver enforces per-image size limits, page count caps, and HTTP timeouts

## Operator checklist

1. Set GitHub secrets: `API_READ_KEY`, `WEBHOOK_SECRET`, `QSTASH_CURRENT_SIGNING_KEY`, `QSTASH_NEXT_SIGNING_KEY`, `PIPELINE_HEALTH_URL`, `KV_REST_API_URL`, `KV_REST_API_TOKEN`
2. Set Vercel env: `NOTIFIER_URL`, `NOTIFIER_API_KEY` (dashboard only)
3. Set repo variable `ALLOWED_ORIGINS` to your dashboard/status origins (comma-separated)
4. Never deploy with `SECURITY_REQUIRE_API_KEY=false` or `SECURITY_REQUIRE_WEBHOOK_AUTH=false` in production
5. Never enable `ADMIN_MUTATIONS_ENABLED` in prod unless you also set a strong `ADMIN_API_KEY`
6. Rotate keys if you suspect exposure; updating GitHub/Vercel secrets is enough — nothing live belongs in git
7. For VM deploys over SSH, set `VM_SSH_KNOWN_HOSTS` in CI to enable strict host key checking

## Residual risks (accepted or follow-up)

- **Cloud Run `allUsers` invoker** — required for QStash/public health today; app-layer auth must stay on
- **Public dashboard proxy** — exposes the same read data as the UI; acceptable for a public watchlist dashboard
- **URL enumeration** — docs use placeholders; live hostnames still discoverable via Vercel/DNS
- **Deploy workflow_dispatch** — restrict via GitHub Environment protection on `production`
- **Terraform state** — may contain secret values; GCP VM bootstrap loads `.env` from Secret Manager instead of instance metadata
- **Grafana Cloud on serverless** — metrics endpoints exist but remote_write requires Alloy sidecar (Compose/VM only)
