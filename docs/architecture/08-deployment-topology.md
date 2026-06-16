# Part 8 — Deployment topology

[← Part 7](./07-security-rationale.md) · [Series index](./README.md) · [Next: Observability →](./09-observability-and-cost.md)

## Decision summary

Compute is **portable** (same OCI images); data plane is **BYO** (Postgres + Kafka or QStash). **Terraform modules** exist for GCP/AWS/Azure/DigitalOcean with three `deployment_target` values. **Configure wizard** generates tier-correct compose/helm. Prod default: **GCP serverless** + **Vercel edge**.

---

## BYO data plane philosophy

| You operate | Project ships |
|-------------|---------------|
| Postgres (Neon, Aiven, RDS, …) | Schema migrations, connection parsing |
| Kafka or QStash | Publisher + consumer wiring |
| Discord/Slack/Telegram | Notifier registry |
| Vercel (optional) | Dashboard + status static assets |
| GCP/AWS/Azure/DO (optional) | Terraform + bootstrap scripts |

**Why not bundled managed Postgres?**

- Operators already have free tiers (Neon, Aiven).
- Avoids billing coupling and data residency arguments.
- Terraform focuses on **compute IAM**, not database vendor choice.

---

## Configure wizard (`configure/`)

### Manifest-first generation

`config/manga-cdc.yaml` records tier decisions; generators emit:

- `.env.example`
- `docker-compose.yml` (local) or prod compose
- Helm values fragments

### Tier enforcement (non-negotiable rules)

| Rule | Reason |
|------|--------|
| Local → embedded Postgres + Redpanda | One-command dev |
| Local → **no QStash/Caddy** | Avoid external account for clone-and-run |
| Production → external DB URL | No Postgres container in prod compose |
| `eventing: qstash` → Caddy in prod artifacts | TLS termination for webhook |

**PE insight:** Tier rules prevent the most common misconfiguration — **Kafka broker pointing at localhost in prod** or **QStash in local compose without docs**.

### Why wizard vs pure documentation?

| Docs only | Wizard (shipped) |
|-----------|------------------|
| Copy-paste errors | Validated manifest |
| Drift between compose and helm | Single source |
| Onboarding friction | `go run ./configure` |

Terraform was **out of wizard v1** intentionally — IaC modules landed separately with their own bootstrap (`scripts/bootstrap.sh`).

---

## Three deployment targets

| Target | Unit of compute | Best for |
|--------|-----------------|----------|
| `vm` | Single Ubuntu VM + Docker Compose | Homelab, full observability stack, Debezium experiments |
| `kubernetes` | Helm on GKE/EKS/AKS/DOKS | Teams with existing K8s |
| `serverless` | Cloud Run / Fargate / ACA / App Platform | Low cost, scale-to-zero |

### Serverless mapping (GCP reference)

```
Cloud Scheduler → Cloud Run Job (scraper, RUN_ONCE)
Cloud Run Service (notifier, HTTP :8080)
Neon Postgres (write + optional read pooler)
QStash OR managed Kafka
Vercel (dashboard + status)
```

**Image registry:** CI pushes to GHCR and Docker Hub; Terraform deploy uses Docker Hub coordinates (`stage-deploy-gcp.yml`).

---

## Terraform multi-cloud

Modules: `terraform/{gcp,aws,azure,digitalocean}/`

Shared patterns:

- Parse `database_url` and `database_read_url` (`postgres://` and `postgresql://`)
- Map to `SPRING_DATASOURCE_*` and read-specific env vars
- `deployment_target` switches resource sets (VM vs GKE vs Cloud Run)
- Sensitive vars marked `sensitive = true`

### VM bootstrap secret handling

| Cloud | Secret delivery |
|-------|-----------------|
| GCP | Secret Manager → startup script |
| AWS | Secrets Manager |
| Azure | Key Vault |
| DO droplet | Inline user_data (weaker — prefer serverless/k8s) |

**ADR:** Never embed production `.env` in instance metadata plaintext on GCP (migrated to Secret Manager).

---

## deploy_method: terraform vs direct

| Method | Use when | Risk |
|--------|----------|------|
| **terraform** | First deploy, env var changes, infra drift | State bucket required; slower |
| **direct** | Image-only tag bump after initial apply | Env drift if manual console edits |

Bootstrap sets `DEPLOY_METHOD=terraform`; operators switch to `direct` for faster iteration.

**Lesson (v0.5.3):** Retag without rebuild can skip deploy while prod runs stale image — see [Part 10](./10-ci-cd-and-release-train.md).

---

## Neon production pattern

| Secret | Role |
|--------|------|
| `DATABASE_URL` | Pooled primary — scraper writes, notifier writes |
| `DATABASE_READ_URL` | Pooled read replica — notifier SELECTs |

Terraform must pass **read credentials explicitly** — JDBC URL alone caused SCRAM auth failures on Cloud Run.

---

## Frontend deploy split

| Asset | Deploy target |
|-------|---------------|
| Dashboard static + Edge API | Vercel project |
| Status static + KV API | Vercel project (separate) |
| `VITE_STATUS_PAGE_URL` | Baked at dashboard **build** time in CI |

**Why build-time URL?** Static client needs absolute link to status page without runtime config service.

---

## Multi-cloud: why four providers?

Not because operators must use all four — **one module per employer/homelab preference**:

| Provider | Typical audience |
|----------|------------------|
| GCP | This repo’s reference prod |
| AWS | Enterprise / Fargate users |
| Azure | ACA shops |
| DigitalOcean | Indie / App Platform |

Code reuse: startup templates, env parsing, scheduler abstractions.

**Cost:** Maintaining four modules — accepted for portfolio + portability.

---

## Failure modes

| Symptom | Cause |
|---------|-------|
| Terraform plan fails on `regex(database_url)` | `postgresql://` vs `postgres://` — fixed in `main.tf` |
| Deploy ok, notifier crash | Missing `SPRING_DATASOURCE_READ_PASSWORD` |
| Scheduler never runs | IAM `run.developer` on scheduler SA |
| Wrong image arch | Local ARM build on Cloud Run — CI builds `linux/amd64` |

---

## Alternatives considered

| Option | Verdict |
|--------|---------|
| Single-cloud Terraform only | Rejected — portability goal |
| Kubernetes-only | Too heavy for default hobby path |
| Fly.io / Railway one-click | Not generalized to four clouds |
| Pulumi vs Terraform | Terraform HCL chosen for module ecosystem |
| GitOps (ArgoCD) | Out of scope Phase 1 |

---

## Change checklist

- [ ] All clouds parse new env vars (or documented GCP-only)?
- [ ] `sensitive = true` on new secret variables?
- [ ] VM bootstrap does not log secrets?
- [ ] Serverless module updates **both** Job and Service?
- [ ] `DATABASE_READ_URL` in deploy workflow env?

---

## Next

- [cloud-setup.md](../cloud-setup.md)
- [terraform/README.md](../../terraform/README.md)
- [Part 9 — Observability & cost](./09-observability-and-cost.md)
