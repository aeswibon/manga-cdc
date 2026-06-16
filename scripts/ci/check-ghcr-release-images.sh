#!/usr/bin/env bash
# Return 0 when all release images exist on ghcr.io for VERSION (e.g. 0.4.5).
# When EXPECTED_COMMIT is set, also require each semver tag digest matches the
# short-SHA tag for that commit (prevents skip_all after retagging a release).
set -euo pipefail

version="${1:?version required (e.g. 0.4.5)}"
expected_commit="${2:-}"

scraper_image="${SCRAPER_IMAGE:-ghcr.io/aeswibon/manga-cdc/scraper}"
notification_image="${NOTIFICATION_IMAGE:-ghcr.io/aeswibon/manga-cdc/notification-service}"
dashboard_image="${DASHBOARD_IMAGE:-ghcr.io/aeswibon/manga-cdc/dashboard}"

manifest_digest() {
  docker buildx imagetools inspect "$1" --format '{{.Manifest.Digest}}' 2>/dev/null
}

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

release_images_exist=false
if [ "$missing" -eq 0 ]; then
  release_images_exist=true
  echo "Release images exist for v${version}"
else
  echo "Release images not published for v${version}"
fi

release_images_match_commit=true
if [ "$release_images_exist" = "true" ] && [ -n "$expected_commit" ]; then
  short_sha="${expected_commit:0:7}"
  echo "Checking release images match commit ${expected_commit} (short ${short_sha})"
  for image in "$scraper_image" "$notification_image" "$dashboard_image"; do
    semver_ref="${image}:${version}"
    sha_ref="${image}:${short_sha}"
    semver_digest="$(manifest_digest "$semver_ref" || true)"
    sha_digest="$(manifest_digest "$sha_ref" || true)"
    if [ -z "$semver_digest" ] || [ -z "$sha_digest" ]; then
      echo "missing digest for ${semver_ref} or ${sha_ref}"
      release_images_match_commit=false
      continue
    fi
    if [ "$semver_digest" != "$sha_digest" ]; then
      echo "digest mismatch: ${semver_ref} (${semver_digest}) vs ${sha_ref} (${sha_digest})"
      release_images_match_commit=false
    else
      echo "digest match ${semver_ref} == ${sha_ref}"
    fi
  done
elif [ -n "$expected_commit" ]; then
  release_images_match_commit=false
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "release_images_exist=${release_images_exist}" >> "$GITHUB_OUTPUT"
  echo "release_images_match_commit=${release_images_match_commit}" >> "$GITHUB_OUTPUT"
fi

if [ "$release_images_exist" = "true" ] && [ "$release_images_match_commit" = "true" ]; then
  exit 0
fi

exit 1
