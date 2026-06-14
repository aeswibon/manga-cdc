#!/usr/bin/env bash
# Move refs/tags/vX.Y.Z to a target commit (default: origin/master).
set -euo pipefail

version="${1:?version required (e.g. 0.5.0)}"
target_sha="${2:-}"

repo="${GITHUB_REPOSITORY:?}"

if [ -z "${RELEASE_BOT_TOKEN:-}" ]; then
  echo "error: RELEASE_BOT_TOKEN is required to update protected refs" >&2
  exit 1
fi

if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "Invalid semver: $version" >&2
  exit 1
fi

gh_api_push() {
  GH_TOKEN="$RELEASE_BOT_TOKEN" gh api "$@"
}

if [ -z "$target_sha" ]; then
  target_sha="$(gh_api_push "repos/${repo}/git/ref/heads/master" --jq .object.sha)"
fi

tag_ref="tags/v${version}"
tag_sha=""
if tag_sha="$(gh_api_push "repos/${repo}/git/ref/${tag_ref}" --jq .object.sha 2>/dev/null)"; then
  :
else
  tag_sha=""
fi

if [ "$tag_sha" = "$target_sha" ]; then
  echo "Tag v${version} already at ${target_sha}"
  exit 0
fi

if [ -n "$tag_sha" ]; then
  gh_api_push "repos/${repo}/git/refs/${tag_ref}" -X PATCH \
    --input - <<< "$(jq -n --arg sha "$target_sha" '{sha: $sha, force: true}')"
else
  gh_api_push "repos/${repo}/git/refs" -X POST \
    --input - <<< "$(jq -n --arg ref "refs/tags/v${version}" --arg sha "$target_sha" '{ref: $ref, sha: $sha}')"
fi

echo "Moved tag v${version} to ${target_sha}"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "tag_moved=true"
    echo "commit_sha=${target_sha}"
  } >> "$GITHUB_OUTPUT"
fi
