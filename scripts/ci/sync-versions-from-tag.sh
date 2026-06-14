#!/usr/bin/env bash
# Align repo version files with a release tag (vX.Y.Z -> X.Y.Z).
set -euo pipefail

usage() {
  echo "Usage: $0 [X.Y.Z]" >&2
  echo "  Reads version from GITHUB_REF=refs/tags/vX.Y.Z when no argument is given." >&2
  exit 1
}

resolve_version() {
  if [[ -n "${1:-}" ]]; then
    echo "$1"
    return
  fi

  local ref="${GITHUB_REF:-}"
  if [[ "$ref" =~ ^refs/tags/v(.+)$ ]]; then
    echo "${BASH_REMATCH[1]}"
    return
  fi

  usage
}

VERSION="$(resolve_version "${1:-}")"
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "Invalid semver: $VERSION" >&2
  exit 1
fi

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

echo "Syncing version files to $VERSION"

perl -i -pe 's/"version": "[^"]+"/"version": "'"$VERSION"'"/' dashboard/package.json
perl -i -pe 's/"version": "[^"]+"/"version": "'"$VERSION"'"/' status-page/package.json

perl -i -pe 's/^version: .+/version: '"$VERSION"'/' helm/manga-cdc/Chart.yaml
perl -i -pe 's/^appVersion: .+/appVersion: "'"$VERSION"'"/' helm/manga-cdc/Chart.yaml

perl -i -pe 's/const Version = "[^"]+"/const Version = "'"$VERSION"'"/' scraper/internal/version/version.go

perl -i -0777 -pe 's/(<artifactId>notification-service<\/artifactId>\s*\n\s*<version>)[^<]+/${1}'"$VERSION"'/s' notification-service/pom.xml

perl -i -pe 's/ARG APP_VERSION=[^\s]+/ARG APP_VERSION='"$VERSION"'/' scraper/Dockerfile notification-service/Dockerfile dashboard/Dockerfile

if git diff --quiet -- \
  dashboard/package.json \
  status-page/package.json \
  helm/manga-cdc/Chart.yaml \
  scraper/internal/version/version.go \
  notification-service/pom.xml \
  scraper/Dockerfile \
  notification-service/Dockerfile \
  dashboard/Dockerfile; then
  echo "Version files already at $VERSION"
else
  echo "Updated version files:"
  git diff --stat -- \
    dashboard/package.json \
    status-page/package.json \
    helm/manga-cdc/Chart.yaml \
    scraper/internal/version/version.go \
    notification-service/pom.xml \
    scraper/Dockerfile \
    notification-service/Dockerfile \
    dashboard/Dockerfile
fi
