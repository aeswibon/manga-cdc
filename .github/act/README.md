# Local GitHub Actions (act)

Run workflows locally with [nektos/act](https://github.com/nektos/act).

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

Expected: version files bump `0.4.4` → `0.4.5`, verify step passes, push skipped (`ACT=true`).

After act, restore local files if needed:

```bash
git checkout -- dashboard/package.json status-page/package.json helm/manga-cdc/Chart.yaml \
  scraper/internal/version/version.go notification-service/pom.xml scraper/Dockerfile notification-service/Dockerfile
```

```bash
bash scripts/ci/sync-versions-from-tag.sh 0.4.5
git diff --stat
git checkout -- dashboard/package.json status-page/package.json helm/manga-cdc/Chart.yaml \
  scraper/internal/version/version.go notification-service/pom.xml scraper/Dockerfile notification-service/Dockerfile
```
