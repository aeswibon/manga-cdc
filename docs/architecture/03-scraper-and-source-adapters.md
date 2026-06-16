# Part 3 — Scraper & source adapters

[← Part 2](./02-system-architecture.md) · [Series index](./README.md) · [Next: Database & eventing →](./04-database-and-eventing.md)

## Decision summary

All **external manga site I/O** lives in a Go **batch scraper** behind a `SourceAdapter` interface. A **diff engine** compares fetched state to Postgres, writes atomically per series, then **optionally publishes** events. Scheduling is **cron-shaped** (Cloud Run Job), not request-driven.

---

## Why a dedicated scraper service?

| Concern | If merged into notifier | Separate scraper |
|---------|-------------------------|------------------|
| CPU spikes during HTML parse | Blocks HTTP threads / GC pauses on API | Isolated Job; API stays responsive |
| IP blocks / retries | Couples notification SLA to scrape retries | Scraper retries without user-visible 503 |
| Release cadence | Force redeploy notifier for parser fix | Independent image tag |
| Horizontal scale | Absurd (duplicate scrapes) | Single scheduler → one Job run |

**PE note:** Scraping is **throughput-oriented batch work**; notification is **latency-sensitive policy + I/O**. Different autoscaler signals (schedule vs HTTP concurrency).

---

## Source adapter contract

Defined in `scraper/internal/adapter/adapter.go`:

```go
type SourceAdapter interface {
    Name() string
    FetchLatest(ctx context.Context) ([]model.Series, error)
    FetchChapters(ctx context.Context, seriesID string) ([]model.Chapter, error)
    FetchPages(ctx context.Context, chapterUrl string) ([]string, error)
}
```

Optional `SeriesMetadataFetcher` for lazy metadata hydration.

### Design choices in the interface

| Method | Purpose | Why on interface |
|--------|---------|------------------|
| `FetchLatest` | Discovery / bulk updates | Some sources expose “recently updated” feeds |
| `FetchChapters` | Per-series chapter list | Watchlist-driven scrape uses stable `source_id` |
| `FetchPages` | Image URLs for future archiver | v0.9 CBZ path without adapter redesign |

**Namespaced IDs:** Watchlist entries use `source:rawId` via `watchlist.NamespacedSourceID` so MangaDex UUIDs cannot collide with MangaPlus IDs in one table.

### Registry pattern

Adapters register in `cmd/scraper/main.go` map — **compile-time registry**, not plugins.

| Approach | Trade-off |
|----------|-----------|
| **Compile-time map** (shipped) | Rebuild to add source; simple; type-safe |
| `plugin.so` dynamic load | Go plugin pain; supply-chain risk |
| External “scraper workers” per source | Six deployables; ops hell |

For six sources and one operator, compile-time registry is correct.

---

## Diff engine (`scraper/internal/diff/`)

### Responsibilities

1. **SyncWatchlist** — reconcile `data/watchlist.yaml` → `manga_series` + `notification_prefs`
2. **ProcessActiveSeries** — per active series: fetch chapters, diff against DB, insert new rows
3. **Validation** — `validate.NormalizeSeries` / chapter rules before write

### Inter-series delay

Default `seriesDelay: 500ms` between series processing — **politeness + rate-limit avoidance**, not a correctness requirement.

**Tunable via** `NewWithDelay` for tests or aggressive homelab runs.

### Correctness properties

| Property | Mechanism |
|----------|-----------|
| No duplicate chapters | `UNIQUE(series_id, chapter_num)` + insert conflict handling |
| Idempotent rescrape | Existing chapters skipped; metadata may update |
| Removed watchlist entries | Deactivate/delete path in sync (cascade on chapters/logs) |
| Prefs drift | YAML `notifications` → JSON stored on sync |

### What diff does *not* do

- Cross-source deduplication (“One Piece on MangaDex == MangaPlus”) — **metadata resolver** is partial; full merge is roadmap.
- Semantic “is this a new arc” — only numeric chapter comparison.

---

## Watchlist: Git as config store

**Decision:** Production watchlist changes via PR to `data/watchlist.yaml`, not dashboard forms.

| Factor | Git PR | In-app CRUD |
|--------|--------|-------------|
| Audit trail | Native (`git log`) | Must build |
| Community contributions | Fork + PR | Auth + abuse |
| Prod attack surface | No admin API | `ADMIN_MUTATIONS_ENABLED` risk |
| Validation | CI `validate-watchlist.py` | Server-side duplicate |

**Operational flow:** Merge PR → next scraper run calls `SyncWatchlist` → DB reflects YAML.

**Trade-off:** Slower time-to-track for non-technical users — accepted until v0.7 linter + possible dashboard editor behind admin key.

---

## Event publish (scraper side)

After insert, scraper may call:

- `kafka.Producer.PublishChapterEvent`
- `qstash.Publisher.PublishChapterEvent`

Both emit the same envelope ([Part 4](./04-database-and-eventing.md)).

**Conditional wiring:** If env vars absent, publisher is `nil` — scraper still writes DB. This supports:

- Local dev without bus
- Serverless with QStash only (no Kafka TCP from Job)

**Failure behavior:** Publish error is logged; chapter remains `is_new=true`. Operator can replay via future tooling or manual webhook — **data not lost**.

---

## Scheduling & runtime

| Profile | Behavior |
|---------|----------|
| Local Compose | Loop + sleep in process |
| GCP serverless | `RUN_ONCE=true` Job; Cloud Scheduler `0 */2 * * *` |
| VM / K8s | CronJob or in-process loop |

### Why Cloud Run Job (not Service) for scraper?

- No HTTP health fiction for a batch worker.
- Bill **per execution second**, not idle time.
- Clear **exit code** for Scheduler retry semantics.

**Constraint:** Job timeout must exceed worst-case full watchlist scrape — watchlist growth increases runtime linearly.

---

## Migrations (goose from Go)

Scraper runs `goose` migrations on startup / dedicated `migrate` CLI.

| Why scraper owns migrations | Why not Flyway in Java only |
|------------------------------|----------------------------|
| Scraper starts first in compose | Notifier would fail if scraper never ran |
| Single migration directory `db/migrations/` | Java tests also need schema |
| Baseline detection for existing DBs | Neon prod may pre-exist |

**Idempotency lesson:** `ADD COLUMN IF NOT EXISTS` for migration 008; e2e runs migrate via docker network because `initdb.d` can re-apply goose Down sections.

---

## Observability

| Signal | Use |
|--------|-----|
| Prometheus `:2112/metrics` | Per-source duration, errors |
| Zero-result detection | Site layout change / block |
| `/healthz`, `/readyz` | Compose/K8s probes |

Serverless Job: metrics are **ephemeral** unless pushed — known gap ([Part 9](./09-observability-and-cost.md)).

---

## Testing strategy

| Layer | Technique | Why |
|-------|-----------|-----|
| Adapter unit | HTML/JSON **fixtures** in repo | Sites flake in CI |
| Diff integration | Real Postgres (testcontainers/docker) | SQL behavior matters |
| E2e | `scripts/e2e.sh` full stack | Catches wiring regressions |

**Senior dev guidance:** Do not mock away parsing edge cases in adapters — add fixture files from saved HTML snapshots.

---

## Failure modes

| Failure | System behavior | Operator action |
|---------|-----------------|-----------------|
| Source HTTP 403/503 | Series error metric; other sources continue | Check block; roadmap: FlareSolverr |
| Parse returns empty | Zero-result alert | Update adapter fixture + parser |
| DB down | Scraper fails readiness; Job retry | Fix Neon connectivity |
| Publish down | DB has `is_new` chapters | Fix QStash/Kafka; notifications catch up |
| Bad watchlist PR | CI rejects | Fix YAML before merge |

---

## Alternatives considered

| Option | Verdict |
|--------|---------|
| Headless Chrome for all sites | Too heavy for Job; selective use later |
| Shared crawler library (colly only) | Site-specific parsers still required |
| Scrape on notifier cron | Violates boundary ([Part 2](./02-system-architecture.md)) |
| Pull-based “notifier asks scraper” | Extra RPC service; push via DB+bus simpler |

---

## v0.6 planned extensions

| Feature | Architectural impact |
|---------|---------------------|
| `fallback_sources` | Adapter chain in diff engine; same DB row |
| Hiatus / stale alerts | Time-based predicates on `last_checked` |
| Schedule hints | Derived from `release_date` history — read-only analytics |

---

## Change checklist

- [ ] New adapter has **fixture tests** with real HTML/API snapshots?
- [ ] `source` key documented in CONTRIBUTING?
- [ ] Rate limits respected (`seriesDelay`, source-specific backoff)?
- [ ] Publish errors do not **rollback** DB insert?
- [ ] Watchlist validator updated for new optional fields?

---

## Next

- [Part 4 — Database & eventing](./04-database-and-eventing.md)
- [CONTRIBUTING.md](../../CONTRIBUTING.md)
