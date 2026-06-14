#!/usr/bin/env bash
# Publish synced version files to protected master.
#
# Tag alignment is handled after a successful release by scripts/ci/move-release-tag.sh
# so moving the tag mid-pipeline does not re-trigger release.yml.
#
# Uses two tokens:
# - GITHUB_TOKEN creates blobs/trees/commits (GitHub-verified Actions signatures)
# - RELEASE_BOT_TOKEN updates protected refs (must be on the master ruleset bypass list)
set -euo pipefail

version="${1:?version required (e.g. 0.4.5)}"
repo="${GITHUB_REPOSITORY:?}"

if [ -z "${RELEASE_BOT_TOKEN:-}" ]; then
  echo "error: RELEASE_BOT_TOKEN is required to update protected refs" >&2
  exit 1
fi

if [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "error: GITHUB_TOKEN is required to create verified commits" >&2
  exit 1
fi

gh_api_content() {
  GH_TOKEN="$GITHUB_TOKEN" gh api "$@"
}

gh_api_push() {
  GH_TOKEN="$RELEASE_BOT_TOKEN" gh api "$@"
}

VERSION_FILES=(
  dashboard/package.json
  status-page/package.json
  helm/manga-cdc/Chart.yaml
  scraper/internal/version/version.go
  notification-service/pom.xml
  scraper/Dockerfile
  notification-service/Dockerfile
  notification-service/Dockerfile.jvm
  dashboard/Dockerfile
)

for path in "${VERSION_FILES[@]}"; do
  if [ ! -f "$path" ]; then
    echo "error: expected version file missing: $path" >&2
    exit 1
  fi
done

master_sha="$(gh_api_push "repos/${repo}/git/ref/heads/master" --jq .object.sha)"
base_tree="$(gh_api_content "repos/${repo}/git/commits/${master_sha}" --jq .tree.sha)"

needs_commit=false
if ! git diff --quiet -- "${VERSION_FILES[@]}"; then
  needs_commit=true
fi

target_sha="$master_sha"

if [ "$needs_commit" = true ]; then
  tree_json='[]'
  for path in "${VERSION_FILES[@]}"; do
    blob_sha="$(gh_api_content "repos/${repo}/git/blobs" \
      --input - <<< "$(jq -n --rawfile content "$path" '{content: $content, encoding: "utf-8"}')" \
      --jq .sha)"
    tree_json="$(echo "$tree_json" | jq --arg p "$path" --arg s "$blob_sha" \
      '. + [{path: $p, mode: "100644", type: "blob", sha: $s}]')"
  done

  tree_sha="$(gh_api_content "repos/${repo}/git/trees" \
    --input - <<< "$(jq -n --arg base "$base_tree" --argjson tree "$tree_json" '{base_tree: $base, tree: $tree}')" \
    --jq .sha)"

  target_sha="$(gh_api_content "repos/${repo}/git/commits" \
    --input - <<< "$(jq -n \
      --arg msg "chore: sync version files to v${version}" \
      --arg tree "$tree_sha" \
      --arg parent "$master_sha" \
      '{
        message: $msg,
        tree: $tree,
        parents: [$parent],
        author: {name: "github-actions[bot]", email: "41898282+github-actions[bot]@users.noreply.github.com"},
        committer: {name: "github-actions[bot]", email: "41898282+github-actions[bot]@users.noreply.github.com"}
      }')" \
    --jq .sha)"

  gh_api_push "repos/${repo}/git/refs/heads/master" -X PATCH \
    --input - <<< "$(jq -n --arg sha "$target_sha" '{sha: $sha, force: false}')" \
    || {
      echo "error: failed to update refs/heads/master." >&2
      echo "Add the RELEASE_BOT_TOKEN actor to the master ruleset bypass list." >&2
      echo "See .github/act/README.md#release-bot-setup" >&2
      exit 1
    }

  echo "Created verified commit ${target_sha} on master"
else
  echo "Version files already aligned on master"
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "pushed_master=${needs_commit}"
    echo "commit_sha=${target_sha}"
  } >> "$GITHUB_OUTPUT"
fi
