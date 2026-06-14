#!/usr/bin/env bash
# Return 0 when all release images exist on ghcr.io for VERSION (e.g. 0.4.5).
set -euo pipefail

version="${1:?version required (e.g. 0.4.5)}"

scraper_image="${SCRAPER_IMAGE:-ghcr.io/aeswibon/manga-cdc/scraper}"
notification_image="${NOTIFICATION_IMAGE:-ghcr.io/aeswibon/manga-cdc/notification-service}"
dashboard_image="${DASHBOARD_IMAGE:-ghcr.io/aeswibon/manga-cdc/dashboard}"

missing=0
for image in "$scraper_image" "$notification_image" "$dashboard_image"; do
  ref="${image}:${version}"
  if docker buildx imagetools inspect "$ref" >/dev/null 2>&1; then
    echo "found ${ref}"
  else
    echo "missing ${ref}"
    missing=1
  fi
done

if [ "$missing" -eq 0 ]; then
  echo "Release images exist for v${version}"
  if [ -n "${GITHUB_OUTPUT:-}" ]; then
    echo "release_images_exist=true" >> "$GITHUB_OUTPUT"
  fi
  exit 0
fi

echo "Release images not published for v${version}"
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "release_images_exist=false" >> "$GITHUB_OUTPUT"
fi
exit 1
