# Part 9 — Observability & cost

[← Part 8](./08-deployment-topology.md) · [Series index](./README.md) · [Next: CI/CD →](./10-ci-cd-and-release-train.md)

## Decision summary

Observability is **tiered**: rich self-hosted metrics locally/VM, **lightweight health + logs** on serverless prod. Cost controls (2h scrape, scale-to-zero, bootstrap scopes, health caches) are **first-class architectural decisions**, not post-hoc tuning.

---

## Observability goals vs SLOs

| Question | Primary signal | SLO (Phase 1, informal) |
|----------|----------------|-------------------------|
| Is pipeline alive? | Status page + `/api/pipeline/health` | Poller every 5 min |
| Are sources working? | Scraper zero-result metrics | Detect within 1 scrape cycle |
| Are notifications delivering? | `notification_log` + dashboard logs | Manual operator review |
| Why did scrape fail? | Cloud Logging / VM Prometheus | Best effort |

**Not promised:** sub-minute chapter detection (2h scheduler), 99.9% uptime on free tiers.

---

## Metrics architecture by tier

| Tier | Scraper metrics | Notifier metrics | Dashboards |
|------|-----------------|------------------|------------|
| Local Compose | Prometheus `:2112` | Actuator/Micrometer | Grafana JSON in repo |
| VM prod | Same + optional Alloy remote_write | Same | Grafana local or Cloud |
| Serverless prod | **Ephemeral** (Job exits) | Actuator; limited remote_write | Status page + logs |

### Grafana Cloud gap (serverless)

`OBSERVABILITY_MODE=grafana-cloud` sets remote_write env vars, but **Cloud Run has no Alloy sidecar** to scrape and forward.

| Workaround | Trade-off |
|------------|-----------|
| Logs-only debugging | Cheaper; less histogram detail |
| Run Alloy on VM | Second deployment |
| Push gateway pattern | Not shipped |

**PE decision:** Accept metrics gap on serverless until operator moves to VM or adds push instrumentation.

---

## Health endpoints (semantics)

| Endpoint | Auth | Depth |
|----------|------|-------|
| `/actuator/health` | Public | JVM up |
| `/api/pipeline/health` | API key | DB ping, scrape freshness, notification summary |

### Pipeline health cache (45s)

`PipelineHealthService` caches `buildHealth()` — synchronized rebuild.

| Without cache | With cache |
|---------------|------------|
| Every status poll hits DB | At most ~1 deep check / 45s / instance |
| Cloud Run bill spikes | Predictable load |

**Trade-off:** Health can be **45s stale** inside notifier — acceptable vs 5 min public poller.

---

## Cost architecture (v0.4+ “balance package”)

Documented in cloud-run dashboard balance design — these are **requirements**, not suggestions:

| Knob | Value | Savings mechanism |
|------|-------|-------------------|
| Scraper cron | `0 */2 * * *` | 12 runs/day vs 96 at */15 |
| Notifier memory | 256 MiB, CPU throttle | Minimum Cloud Run SKU |
| `CDC_ENABLED=false` (QStash path) | No Kafka consumer threads | Smaller steady-state CPU |
| Dashboard poll | 2 min, visible tab only | Fewer proxy invocations |
| Bootstrap scopes | Split overview/watchlist | Smaller payloads |
| Edge cache | 30s + SWR | Dedup concurrent readers |
| SSE in prod | Off | No long-lived connections |
| Warm pings | **Not used** | Would defeat scale-to-zero |

### Scale-to-zero behavior

Cloud Run notifier scales to zero when idle.

| Effect | Mitigation |
|--------|------------|
| Cold start on first dashboard visit | Accept 2–5s latency |
| JDBC pool cold | Hikari fail-fast; readiness may flap briefly |
| Operator annoyance | Cheaper than min-instances=1 |

**When to set `min-instances=1`:** Operator explicitly values latency over ~$month — document in personal Terraform vars, not repo default.

---

## Scraper operability signals

| Metric / alert | Meaning |
|----------------|---------|
| Per-source scrape duration | Slow source or network |
| Zero results | Parser broken or IP block |
| Job failure exit code | Scheduler retry; check logs |

On serverless, **export metrics to logging** if Prometheus unavailable — consider structured log counters in future.

---

## Notification audit trail

`notification_log` table is **durable observability** independent of log retention:

| Column | Ops use |
|--------|---------|
| `status` | SENT vs FAILED |
| `channel` | Which notifier |
| `error_message` | Discord 429, etc. |

Dashboard logs tab surfaces recent rows — no Loki required for hobby debugging.

---

## Failure modes

| Symptom | Likely cause |
|---------|--------------|
| Status green, no notifications | Channel env wrong — check `notification_log` |
| Grafana empty on serverless | Expected — no scraper |
| Health flaps 503 on deploy | Read replica creds / cold start |
| High Neon bill | Dashboard poll too aggressive — check Edge cache |
| Scraper “succeeds” but no data | Zero-result not alerting — check metrics |

---

## Alternatives considered

| Option | Verdict |
|--------|---------|
| OpenTelemetry everywhere | Heavier agents on Job |
| 15-minute scrape | Fresher data; 4× Cloud Run Job cost |
| Always-on notifier `min=1` | Rejected as default |
| Client-side direct notifier polling | Key exposure |
| Drop Grafana entirely | Keep for VM/homelab education |

---

## Change checklist

- [ ] New poll loop respects `visibilityState`?
- [ ] New health check behind cache?
- [ ] Serverless change assessed for **connection hold**?
- [ ] Metrics path documented if not available serverless?

---

## Next

- [Part 6 — Operator surfaces](./06-operator-surfaces.md)
- [Part 10 — CI/CD](./10-ci-cd-and-release-train.md)
