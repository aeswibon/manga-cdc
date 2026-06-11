#!/usr/bin/env bash
# Run docker-snapshot + e2e-test locally via act (OrbStack/Colima compatible).
set -euo pipefail

cd "$(dirname "$0")/.."

if [ -z "${DOCKER_HOST:-}" ]; then
  for sock in \
    "${HOME}/.orbstack/run/docker.sock" \
    "${HOME}/.colima/default/docker.sock" \
    "/var/run/docker.sock"; do
    if [ -S "$sock" ]; then
      export DOCKER_HOST="unix://${sock}"
      break
    fi
  done
fi

if [ -z "${DOCKER_HOST:-}" ]; then
  echo "error: set DOCKER_HOST to your Docker socket (OrbStack/Colima/Docker Desktop)" >&2
  exit 1
fi

echo "Using DOCKER_HOST=${DOCKER_HOST}"

TOKEN="${GITHUB_TOKEN:-$(gh auth token 2>/dev/null || true)}"
if [ -z "$TOKEN" ]; then
  echo "warning: no GITHUB_TOKEN; docker-snapshot push to GHCR may fail" >&2
  TOKEN="act-local"
fi

exec act push \
  -W .github/workflows/test-and-build.yml \
  -j docker-snapshot \
  -j e2e-test \
  --container-architecture linux/amd64 \
  --container-daemon-socket - \
  -s "GITHUB_TOKEN=${TOKEN}" \
  "$@"
