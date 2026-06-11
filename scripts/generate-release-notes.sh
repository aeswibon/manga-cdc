#!/usr/bin/env bash
# Generate markdown release notes for a version tag.
set -euo pipefail

VERSION="${1:-${GITHUB_REF_NAME:-}}"
if [ -z "$VERSION" ]; then
  echo "usage: $0 <version-tag> [previous-tag]" >&2
  exit 1
fi

PREV_TAG="${2:-}"
if [ -z "$PREV_TAG" ]; then
  PREV_TAG="$(git tag --sort=-v:refname | grep -E '^v[0-9]' | grep -v "^${VERSION}$" | head -1 || true)"
fi

SEMVER="${VERSION#v}"

cat <<EOF
# ${VERSION}

EOF

if [ -n "$PREV_TAG" ]; then
  echo "## What's changed"
  echo ""
  git log "${PREV_TAG}..${VERSION}" --pretty=format:'- %s (%h)' --no-merges || true
  echo ""
  echo ""
else
  echo "Initial tagged release."
  echo ""
fi

cat <<EOF
## Container images

| Service | Image |
|---------|-------|
| scraper | \`ghcr.io/aeswibon/manga-cdc/scraper:${SEMVER}\` |
| notification-service | \`ghcr.io/aeswibon/manga-cdc/notification-service:${SEMVER}\` |

Also tagged as \`:${SEMVER%.*}\` and \`:${SEMVER%%.*}\` (semver minor/major aliases).

## Deploy

Production deploy resolves the semver tag automatically when this release tag is present on the commit.

\`\`\`bash
docker pull ghcr.io/aeswibon/manga-cdc/scraper:${SEMVER}
docker pull ghcr.io/aeswibon/manga-cdc/notification-service:${SEMVER}
\`\`\`
EOF
