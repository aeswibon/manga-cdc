# Part 6 — Operator surfaces

[← Part 5](./05-notification-service.md) · [Series index](./README.md) · [Next: Security →](./07-security-rationale.md)

## Decision summary

Operator UI is a **static Svelte app on Vercel** with **Edge API routes** acting as a **read-only BFF**. Secrets stay server-side. **Bootstrap** endpoints reduce fan-out. **Status page** reads **KV** populated by a **GitHub poller** — browsers never call the notifier for deep health.

---

## Two apps, two SLOs

| App | SLO intent | Failure impact |
|-----|------------|----------------|
| **Dashboard** | Fresh enough for humans (~2 min) | Readers see stale chapter list |
| **Status page** | Honest pipeline health (~5 min) | Public embarrassment if wrong |

Coupling them in one deployable would mix **cache TTLs**, **auth models**, and **release cadence**.

---

## Dashboard: why Vercel + Edge proxy?

### Problem statement

Browsers cannot hold `NOTIFIER_API_KEY`. Options:

| Approach | PE assessment |
|----------|---------------|
| API key in `VITE_*` env | **Unacceptable** — key in client bundle |
| Cognito / OAuth for readers | Phase 2 scope |
| **Server-side proxy on Vercel** (shipped) | Key in Edge runtime only; public dashboard URL OK |
| Notifier CORS + public reads | Harvestable data; no key rotation story |

### Proxy implementation (`dashboard/api/data/_proxy.ts`)

**Allowlisted GET paths only:**

```
/api/stats, /api/series, /api/logs, /api/logs/stream
```

**Security properties:**

- Injects `X-Api-Key` from `NOTIFIER_API_KEY` server env
- Strips client `X-Api-Key`, `Authorization`, webhook headers
- Hop-by-hop headers removed on response
- In-memory Edge cache (`UPSTREAM_CACHE_TTL_MS` ~30s) with `stale-while-revalidate`

**Accepted risk:** Anyone can call `/api/data/series` on Vercel — same as using the UI. This is a **product choice** for a public community watchlist, not a leak.

### Why not SvelteKit SSR on Cloud Run?

| Vercel Edge | Cloud Run dashboard |
|-------------|---------------------|
| Global CDN for static assets | Single region |
| Zero ops for TLS | Another service to scale |
| Fits hobby spend | Extra always-on cost |

Dashboard **image** is still built in CI for optional Cloud Run deploy — Vercel is primary prod path.

---

## Bootstrap API (`dashboard/api/data/bootstrap.ts`)

### Problem

Overview tab needed stats + logs; watchlist needed series. Naïve client fetched **everything on load** → large JSON, slow cold start on notifier, expensive on Neon read pool.

### Decision

| `?scope=` | Fetches | Typical size |
|-----------|---------|--------------|
| `overview` | stats + 5 logs | Small |
| `watchlist` | series only | Medium |
| `full` | stats + series + 20 logs | Large (compat) |

### Client behavior (`App.svelte`)

- Poll every **2 minutes** when `document.visibilityState === 'visible'`
- **No polling** when tab hidden — critical for Cloud Run scale-to-zero
- Watchlist tab triggers scope fetch on first visit

**PE trade-off:** Data can be **2 min stale** — acceptable vs always-on notifier cost.

---

## Cover proxy (`dashboard/api/cover.ts`)

External CDN images (MangaDex, MangaPlus) hit **hotlink protection and ad blockers**.

Edge proxy:

- Allowlisted hostnames only
- Validated redirects
- Prevents arbitrary URL fetch (SSRF class)

---

## Status page: KV indirection

### Problem

Public status needs DB/Kafka freshness signals. Exposing `/api/pipeline/health` to browsers either:

- Leaks infra details, or
- Requires API key in client

### Architecture

```
GitHub Actions (health-poller.yml, */5 min)
  → GET PIPELINE_HEALTH_URL + X-Api-Key
  → sanitize response
  → WRITE Vercel KV

Browser → GET /api/status (Edge) → READ KV only
```

**Properties:**

- Notifier URL and API key never in client bundle
- Sanitized JSON — no raw SQL exception text
- Stale KV → status page shows degraded (operator must fix poller)

### Why not Vercel Cron instead of GitHub Actions?

| GHA poller (shipped) | Vercel Cron |
|----------------------|-------------|
| Secrets already in GitHub for deploy | Duplicate secret surface |
| Same trust zone as release train | Extra Vercel enterprise features |

Either works — we standardized on **secrets colocation with CI**.

---

## PWA & update banner

Service worker caches shell; **update banner** prompts reload when new build deployed.

**Why:** Prevents week-old JS calling new API shapes after rapid releases — reduces “works in incognito” support burden.

---

## Caching stack (multi-tier)

| Tier | TTL | Location |
|------|-----|----------|
| Edge bootstrap cache | ~30s | Vercel function memory |
| Pipeline health | 45s | Notifier JVM (`PipelineHealthService`) |
| Status KV | ~5 min | Written by poller |
| Client poll | 2 min | Browser timer |

**PE guidance:** Do not add aggressive 5s polling without revisiting Cloud Run cost model ([Part 9](./09-observability-and-cost.md)).

---

## Failure modes

| User-visible | Root cause |
|--------------|------------|
| Dashboard 502 `{ error: "... HTTP 500" }` | Notifier/DB — check Cloud Run logs |
| Empty watchlist | Bootstrap scope error or API key |
| Status “stale” | Poller failed; KV not updated |
| Covers broken | CDN block; check allowlist |
| Fast battery drain (mobile) | Should not happen — hidden tab stops poll |

---

## Alternatives considered

| Option | Verdict |
|--------|---------|
| SSE to browser in prod | Keeps notifier warm — rejected for serverless |
| tRPC / GraphQL BFF | REST bootstrap sufficient |
| Embed Grafana public dashboard | Too technical for readers |
| Single combined domain | Split cache/security policies |

---

## Change checklist

- [ ] New proxy route **GET-only** and allowlisted?
- [ ] Bootstrap scope documented for client?
- [ ] Cache TTL impact on freshness documented?
- [ ] No secrets in `VITE_*` client env?
- [ ] Status poller updated if health JSON shape changes?

---

## Next

- [Part 7 — Security](./07-security-rationale.md)
- [Part 9 — Observability & cost](./09-observability-and-cost.md)
- [status-page/README.md](../../status-page/README.md)
