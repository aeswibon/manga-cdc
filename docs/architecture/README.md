# Architecture rationale — reading guide

This series explains **why** manga-cdc is built the way it is. It is written for **senior engineers and principal engineers** who need to evaluate, extend, or operate the system — not only *what* exists, but *what we refused to build yet*, *what breaks if you change it*, and *how Phase 2/3 migrations stay open*.

Operational runbooks live in [cloud-setup.md](../cloud-setup.md) and [security-model.md](../security-model.md). This is the **design record**.

## Audience

| Reader | Use this series to… |
|--------|---------------------|
| **Senior developer** | Onboard on invariants, extension points, and coupling between scraper / DB / bus / notifier |
| **Principal engineer** | Challenge trade-offs, assess blast radius of changes, plan Phase 2 SaaS or true CDC |
| **Operator** | Understand why prod is configured the way it is (serverless cost profile, read replica, release gates) |
| **Contributor** | Know where to add adapters, prefs, or UI without violating trust boundaries |

## How to read

| Part | Title | PE-level focus |
|------|--------|----------------|
| [01 — Problem & goals](./01-problem-and-goals.md) | Charter & constraints | Scope boundaries, phased risk budget |
| [02 — System architecture](./02-system-architecture.md) | Boundaries & coupling | Service split, consistency model, extension seams |
| [03 — Scraper & sources](./03-scraper-and-source-adapters.md) | Ingestion design | Adapter contract, diff correctness, scrape SLOs |
| [04 — Database & eventing](./04-database-and-eventing.md) | Durability & delivery | Dual-write semantics, idempotency, CDC migration |
| [05 — Notification service](./05-notification-service.md) | Policy & delivery | Filter graph, auth model, read/write split |
| [06 — Operator surfaces](./06-operator-surfaces.md) | Edge & BFF | Cache tiers, secret handling, cost of freshness |
| [07 — Security rationale](./07-security-rationale.md) | Threat model | Accepted risks, rotation, fail-closed defaults |
| [08 — Deployment topology](./08-deployment-topology.md) | Portability | BYO data plane, Terraform vs direct, multi-cloud |
| [09 — Observability & cost](./09-observability-and-cost.md) | SLO vs spend | Scale-to-zero, health cache, metric gaps |
| [10 — CI/CD & release train](./10-ci-cd-and-release-train.md) | Change safety | Pipeline DAG, skip gates, e2e contract |
| [11 — Roadmap & evolution](./11-roadmap-and-evolution.md) | Future ADRs | WAL CDC, SaaS, alternatives index |

**Suggested path:** 01 → 02 (mandatory for any architectural change) → layer-specific parts → 07 if touching auth → 10 if touching release.

## Conventions

| Term | Meaning |
|------|---------|
| **Shipped** | On `master` and intended for production use |
| **Planned** | On the release train with a written plan |
| **Deferred** | Evaluated; rejected or postponed with documented reason |
| **ADR** | Architecture Decision Record — decision + context + consequences |

Each part includes:

- **Decision summary** — one paragraph a PE can forward in a design review
- **Invariants** — rules that must not break silently
- **Failure modes** — what operators see when things go wrong
- **Alternatives considered** — credible options we did not pick
- **Change checklist** — questions to ask before merging a PR in that area

## Related docs

| Document | Purpose |
|----------|---------|
| [README.md](../../README.md) | Quick start, layout |
| [CONTRIBUTING.md](../../CONTRIBUTING.md) | Watchlist PR workflow |
| [security-model.md](../security-model.md) | Secret checklist |
| [cloud-setup.md](../cloud-setup.md) | Provider bootstrap |
| [terraform/README.md](../../terraform/README.md) | IaC variables |
