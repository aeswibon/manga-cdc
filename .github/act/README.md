# Local GitHub Actions (act)

Run workflows locally with [nektos/act](https://github.com/nektos/act).

## CI pipeline order

| Trigger | Workflows |
|---|---|
| Push to `master` (normal commit) | **Test and Build** only |
| Push tag `v*` | **Sync version from tag** → **Test and Build** (master, release) → **Deploy** |

Tag pushes no longer start Test and Build directly. Sync updates version files on `master`, retags, then the master push runs the release build. Deploy runs after a successful release Test and Build run.

## Release bot setup

The workflow uses two tokens:

| Token | Role |
|---|---|
| `GITHUB_TOKEN` (automatic) | Creates blobs/trees/commits with GitHub-verified Actions signatures |
| **`RELEASE_BOT_TOKEN`** (secret) | Updates protected `master` and release tags; actor must be on the ruleset **bypass list** |

Create a **fine-grained PAT** with **Contents: Read and write** on this repository, add **your GitHub user** (the PAT owner) to the `master` ruleset **Bypass list**, then store the PAT as **`RELEASE_BOT_TOKEN`**.

Without bypass access you will see `GH013` / HTTP 422 (PR required or unverified signature on ref update).

## Prerequisites

- Docker (OrbStack/Colima) running
- `act` installed

This repo's `.actrc` sets `--container-daemon-socket -`, which breaks on macOS. Pass the OrbStack socket explicitly:

```bash
export ACT_DOCKER_SOCKET="${HOME}/.orbstack/run/docker.sock"
# or Colima: export ACT_DOCKER_SOCKET="${HOME}/.colima/default/docker.sock"
```

## Sync version from tag

Event fixture: `.github/act/tag-push-v0.4.5.json`

```bash
ACT=true act push --bind \
  -W .github/workflows/sync-version-on-tag.yml \
  -e .github/act/tag-push-v0.4.5.json \
  -j sync-version \
  --container-daemon-socket "unix://${ACT_DOCKER_SOCKET}"
```

Use `--bind` so act sees new/uncommitted workflow files. Without `--bind`, only tracked files are copied into the container.

Expected: version files bump `0.4.4` → `0.4.5`, verify step passes, push/retag skipped (`ACT=true`).

After act, restore local files if needed:

```bash
git checkout -- dashboard/package.json status-page/package.json helm/manga-cdc/Chart.yaml \
  scraper/internal/version/version.go notification-service/pom.xml scraper/Dockerfile notification-service/Dockerfile
```

Script-only smoke test:

```bash
bash scripts/ci/sync-versions-from-tag.sh 0.4.5
git diff --stat
git checkout -- dashboard/package.json status-page/package.json helm/manga-cdc/Chart.yaml \
  scraper/internal/version/version.go notification-service/pom.xml scraper/Dockerfile notification-service/Dockerfile
```
