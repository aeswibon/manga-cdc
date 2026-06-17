#!/usr/bin/env bash
# Wait until release images are pullable on Docker Hub before Terraform/Cloud Run deploy.
# Docker Hub tags can lag GHCR imagetools pushes by a minute or two.
set -euo pipefail

version="${1:?version required (e.g. 0.6.1)}"

scraper_image="${SCRAPER_IMAGE:?SCRAPER_IMAGE required}"
notification_image="${NOTIFICATION_IMAGE:?NOTIFICATION_IMAGE required}"

max_attempts="${DOCKERHUB_WAIT_MAX_ATTEMPTS:-30}"
sleep_seconds="${DOCKERHUB_WAIT_SLEEP_SECONDS:-10}"

image_exists() {
  docker buildx imagetools inspect "$1" >/dev/null 2>&1
}

wait_for() {
  local ref="$1"
  local attempt=1

  while [ "$attempt" -le "$max_attempts" ]; do
    if image_exists "$ref"; then
      echo "found ${ref} (attempt ${attempt})"
      return 0
    fi

    echo "waiting for ${ref} (attempt ${attempt}/${max_attempts})..."
    if [ "$attempt" -eq "$max_attempts" ]; then
      echo "timeout waiting for ${ref}" >&2
      return 1
    fi

    sleep "$sleep_seconds"
    attempt=$((attempt + 1))
  done
}

for image in "$scraper_image" "$notification_image"; do
  wait_for "${image}:${version}"
done

echo "Docker Hub release images ready for v${version}"
