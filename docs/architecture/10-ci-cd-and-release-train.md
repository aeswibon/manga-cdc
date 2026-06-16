# Part 10 — CI/CD & release train

[← Part 9](./09-observability-and-cost.md) · [Series index](./README.md) · [Next: Roadmap →](./11-roadmap-and-evolution.md)

## Decision summary

**`master` runs tests; tags deploy prod.** Release orchestration uses **pipeline-compose v2** with staged workflows, artifact handoff, and e2e on **snapshot images** before release tags push to registries. **Version-sync** can skip the entire train — operators must understand **green workflow ≠ deployed fix**.

---

## Branch vs tag philosophy

| Trigger | Pipeline | Rationale |
|---------|----------|-----------|
| PR / `master` | Unit tests, terraform validate, lint | Fast feedback |
| Tag `v*.*.*` | Full train + deploy | Explicit operator consent to prod change |

**Why not deploy every merge?**

- Protected `master` + community PRs — blast radius too large
- Hobby prod — operator may merge docs without wanting release
- Tagged semver communicates **support and rollback point**

---

## Pipeline-compose DAG

`.github/pipelines/release.yml`:

```
version-sync
    ├─► test (unit)
    ├─► terraform (validate matrix)
    └─► docker-build (snapshot)
            └─► test-e2e
                    └─► docker-release
                            ├─► github-release
                            ├─► deploy-gcp
                            └─► deploy-vercel (after gcp)
```

Companion wrapper: `.github/workflows/release.yml` runs `aeswibon/pipeline-compose-run@v1.3.0`.

### Why pipeline-compose?

| Monolithic workflow | Staged compose (shipped) |
|---------------------|--------------------------|
| 2000-line YAML | Reusable `stage-*.yml` |
| Hard to retry one stage | Dispatch individual stages |
| Opaque artifact passing | `pipeline-compose-export` outputs |

**Migration note:** v1 schema rejected by compose 1.0.0 — upgraded to `version: 2` + `pipelines:` map (June 2026).

---

## Stage responsibilities (PE detail)

| Stage | Guarantees | Does not guarantee |
|-------|------------|-------------------|
| **version-sync** | Version files match tag; computes `skip_all` | Pushing fix to prod |
| **test** | Go/Java/dashboard unit tests pass | Production config correct |
| **terraform** | HCL validates all clouds | `terraform apply` |
| **docker-build** | Images build for `linux/amd64` | Security scan (unless added) |
| **test-e2e** | Compose stack healthy with snapshots | Live site scraping |
| **docker-release** | Semver tags on GHCR + Docker Hub | Vercel promote |
| **deploy-gcp** | Terraform apply or direct run update | Neon migrations applied |
| **deploy-vercel** | Dashboard + status deploy | DNS/custom domain |

---

## `skip_all` gate (critical ops knowledge)

`stage-version-sync.yml` sets `skip_all=true` when:

1. Version files already synced on `master`
2. GHCR release images exist for semver
3. Tag commit == `master` HEAD

→ Entire downstream pipeline **skipped**; release workflow still **green**.

### v0.5.3 incident pattern

| What happened | Why |
|---------------|-----|
| Tag retagged to new commit | Images already `0.5.3` in GHCR |
| `skip_all=true` | No rebuild, no deploy |
| Prod on manual `0.5.3-neon-fix` image | Operator hotfix outside train |

### Mitigations

| Action | When |
|--------|------|
| Patch bump `v0.5.4` | Need full train |
| Manual `workflow_dispatch` deploy | Hotfix image/env |
| Force rebuild (delete GHCR semver tag) | Same version, new bytes — use carefully |
| Future: `skip_all` considers image digest vs tag commit | Engineering backlog |

**PE rule:** Treat **release workflow success as necessary, not sufficient** for prod state.

---

## E2E contract (`scripts/e2e.sh`)

Runs against CI-built images — not host `go run`.

| Step | Why |
|------|-----|
| Docker network migrate | Avoid host `localhost:5432` flake |
| goose via migrate container | `initdb.d` may run goose Down sections |
| `set -e`-safe conditionals | Bash pitfall fixed (`if` under `set -e`) |
| Health + pipeline smoke | Catches JDBC, proxy wiring |

**E2e does not:** scrape live MangaDex — adapter unit tests do.

---

## Signing & protected branch

- **GPG-signed commits** (repo hook)
- **`RELEASE_BOT_TOKEN`** pushes version-sync to protected `master`
- **Signed tags** for releases

Bypass rules on `master` exist for release bot — document in `.github/act/README.md`.

---

## Deploy secrets wiring

`stage-deploy-gcp.yml` Terraform apply env includes:

- `DATABASE_URL`, `DATABASE_READ_URL` (read replica — must not be omitted)
- Webhook and API keys
- `TF_STATE_BUCKET`

Missing `DATABASE_READ_URL` in workflow caused prod deploy without read creds until fixed (`33238aa`).

---

## Testing matrix (what CI proves)

| Layer | Tool | Gap |
|-------|------|-----|
| Scraper | `go test` | Live site drift |
| Notifier | `mvn test` | Some tests need Postgres |
| Dashboard | `bun test` + build | Visual regressions |
| Watchlist | `validate-watchlist.py` | Semantic title dedup edge cases |
| Terraform | `validate` | Not full apply in PR |

---

## Failure modes

| CI green, prod bad | Cause |
|--------------------|-------|
| skip_all | No deploy |
| E2e passed, prod secrets wrong | Manual env drift |
| Image arch wrong | Built on ARM locally, not CI |
| Terraform apply old workflow file | Dispatch before push landed |

---

## Alternatives considered

| Option | Verdict |
|--------|---------|
| Continuous deploy | Rejected |
| ArgoCD GitOps | Phase 2+ complexity |
| Blue/green Cloud Run | Not shipped |
| Single workflow file | Replaced by compose |
| Manual only releases | Too error-prone |

---

## Change checklist

- [ ] New stage added to `release.yml` with `needs` / `when`?
- [ ] `pipeline-compose-export` outputs documented?
- [ ] E2e covers new wiring?
- [ ] Does change require **skip_all** logic update?
- [ ] Deploy workflow passes new secrets?

---

## Next

- [Part 11 — Roadmap](./11-roadmap-and-evolution.md)
- [.github/act/README.md](../../.github/act/README.md)
