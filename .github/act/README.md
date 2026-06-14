# Local GitHub Actions (act)

Run workflows locally with [nektos/act](https://github.com/nektos/act).

## CI pipeline order

| Trigger | Pipeline |
|---|---|
| Pull request | **Pull request** → `test` → `terraform` → `docker-build` → `test-e2e` via [pipeline-compose-run](https://github.com/aeswibon/pipeline-compose-run) |
| Push tag `v*` | **Release** → `version-sync` → test/tf/build stages → `deploy-gcp` → `deploy-vercel` → `finalize-release` |
| Manual deploy | **Deploy (manual)** → selected cloud + Vercel |

Stage workflows live under `.github/workflows/stage-*.yml`. Orchestration lives in `.github/pipelines/`.

### Single release run per tag push

`version-sync` commits version file bumps to `master` but **does not move the release tag** mid-pipeline (that would fire a second `release.yml`). After deploy succeeds, `finalize-release`:

1. Creates an empty **anchor commit** on `master` with `[skip release] [skip ci]` in the message
2. Moves `vX.Y.Z` to that anchor commit

GitHub skips workflows for commits with `[skip ci]`, so the tag realign does not start another Release run. `release-gate.sh` is a backup: it also skips the pipeline when the tagged commit contains `[skip release]`, or when a tag **update** points at `master` and release images already exist.

## Release bot setup

The workflow uses two tokens:

| Token | Role |
|---|---|
| `GITHUB_TOKEN` (automatic) | Creates blobs/trees/commits with GitHub-verified Actions signatures |
| **`RELEASE_BOT_TOKEN`** (secret) | Updates protected `master` and release tags; actor must be on the ruleset **bypass list** |

Create a **fine-grained PAT** with **Contents: Read and write** on this repository, add **your GitHub user** (the PAT owner) to the **`master`** ruleset **Bypass list**, then store the PAT as **`RELEASE_BOT_TOKEN`**.

Also ensure these rulesets allow bypass for the release bot (typically **Repository admin** role):

| Ruleset | Why |
|---|---|
| `master` | Protected branch / PR / status checks |
| `Require Signed Commits (Branches)` | Verified signatures on all branches |
| `release-tag` | Verified signatures on `v*` tags |

Without bypass on all applicable rulesets you may see `GH013` or HTTP 422 on master or tag updates.

## Prerequisites

- Docker (OrbStack/Colima) running
- `act` installed

This repo's `.actrc` sets `--container-daemon-socket -`, which breaks on macOS. Pass the OrbStack socket explicitly:

```bash
export ACT_DOCKER_SOCKET="${HOME}/.orbstack/run/docker.sock"
# or Colima: export ACT_DOCKER_SOCKET="${HOME}/.colima/default/docker.sock"
```

## Version sync stage

Prefer the script-only smoke test below for local version bumps. Running the full release pipeline with act hits GitHub (checkout, GHCR, publish) and requires `-s RELEASE_BOT_TOKEN` plus a valid `GITHUB_TOKEN`.

Event fixture: `.github/act/tag-push-v0.4.5.json`

```bash
act workflow_dispatch --bind \
  -W .github/workflows/stage-version-sync.yml \
  -e .github/act/tag-push-v0.4.5.json \
  -j sync-version \
  -s RELEASE_BOT_TOKEN=... \
  -s GITHUB_TOKEN=... \
  --container-daemon-socket "unix://${ACT_DOCKER_SOCKET}"
```

Use `--bind` so act sees new/uncommitted workflow files. Without `--bind`, only tracked files are copied into the container.

Script-only smoke test (recommended locally):

```bash
bash scripts/ci/sync-versions-from-tag.sh 0.4.5
git diff --stat
git checkout -- dashboard/package.json status-page/package.json helm/manga-cdc/Chart.yaml \
  scraper/internal/version/version.go notification-service/pom.xml scraper/Dockerfile \
  notification-service/Dockerfile.jvm dashboard/Dockerfile
```
