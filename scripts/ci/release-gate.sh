#!/usr/bin/env bash
# Decide whether the Release workflow should run pipeline-compose.
#
# Writes proceed=true|false and reason=... to GITHUB_OUTPUT when set.
set -euo pipefail

ref="${GITHUB_REF:-}"
if ! [[ "$ref" =~ ^refs/tags/v(.+)$ ]]; then
  echo "error: expected refs/tags/vX.Y.Z, got ${ref:-<empty>}" >&2
  exit 1
fi
version="${BASH_REMATCH[1]}"

proceed=true
reason="normal"

commit_message="$(git log -1 --format=%B)"
if grep -qF '[skip release]' <<< "$commit_message"; then
  proceed=false
  reason="skip_release_marker"
  echo "Release gate: tagged commit contains [skip release]; skipping pipeline."
elif [ "${GITHUB_EVENT_CREATED:-true}" = "false" ]; then
  git fetch origin master "refs/tags/v${version}"
  master_sha="$(git rev-parse origin/master)"
  tag_sha="$(git rev-parse HEAD)"
  if [ "$tag_sha" = "$master_sha" ]; then
    set +e
    bash scripts/ci/check-ghcr-release-images.sh "${version}"
    images_ok=$?
    set -e
    if [ "$images_ok" -eq 0 ]; then
      proceed=false
      reason="tag_realigned_already_released"
      echo "Release gate: tag update points at master with published images; skipping pipeline."
    fi
  fi
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "proceed=${proceed}" >> "$GITHUB_OUTPUT"
  echo "reason=${reason}" >> "$GITHUB_OUTPUT"
fi

if [ "$proceed" = "true" ]; then
  echo "Release gate: proceeding with pipeline (reason=${reason})."
else
  echo "Release gate: skipping pipeline (reason=${reason})."
fi
