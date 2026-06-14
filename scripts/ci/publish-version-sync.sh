#!/usr/bin/env bash
# Publish synced version files to protected master and move the release tag via GitHub API.
# Requires RELEASE_BOT_TOKEN: a PAT or GitHub App token on the repository ruleset bypass list.
set -euo pipefail

version="${1:?version required (e.g. 0.4.5)}"
repo="${GITHUB_REPOSITORY:?}"

if [ -n "${RELEASE_BOT_TOKEN:-}" ]; then
  export GH_TOKEN="$RELEASE_BOT_TOKEN"
elif [ -n "${GITHUB_TOKEN:-}" ]; then
  export GH_TOKEN="$GITHUB_TOKEN"
  echo "::warning::RELEASE_BOT_TOKEN is not set; push to protected master will likely fail (GH013)."
else
  echo "error: RELEASE_BOT_TOKEN or GITHUB_TOKEN is required" >&2
  exit 1
fi

VERSION_FILES=(
  dashboard/package.json
  status-page/package.json
  helm/manga-cdc/Chart.yaml
  scraper/internal/version/version.go
  notification-service/pom.xml
  scraper/Dockerfile
  notification-service/Dockerfile
)

for path in "${VERSION_FILES[@]}"; do
  if [ ! -f "$path" ]; then
    echo "error: expected version file missing: $path" >&2
    exit 1
  fi
done

master_sha="$(gh api "repos/${repo}/git/ref/heads/master" --jq .object.sha)"
base_tree="$(gh api "repos/${repo}/git/commits/${master_sha}" --jq .tree.sha)"

tag_ref="tags/v${version}"
tag_sha=""
if tag_sha="$(gh api "repos/${repo}/git/ref/${tag_ref}" --jq .object.sha 2>/dev/null)"; then
  :
else
  tag_sha=""
fi

needs_commit=false
if ! git diff --quiet -- "${VERSION_FILES[@]}"; then
  needs_commit=true
fi

target_sha="$master_sha"

if [ "$needs_commit" = true ]; then
  tree_json='[]'
  for path in "${VERSION_FILES[@]}"; do
    blob_sha="$(gh api "repos/${repo}/git/blobs" \
      --input - <<< "$(jq -n --rawfile content "$path" '{content: $content, encoding: "utf-8"}')" \
      --jq .sha)"
    tree_json="$(echo "$tree_json" | jq --arg p "$path" --arg s "$blob_sha" \
      '. + [{path: $p, mode: "100644", type: "blob", sha: $s}]')"
  done

  tree_sha="$(gh api "repos/${repo}/git/trees" \
    --input - <<< "$(jq -n --arg base "$base_tree" --argjson tree "$tree_json" '{base_tree: $base, tree: $tree}')" \
    --jq .sha)"

  target_sha="$(gh api "repos/${repo}/git/commits" \
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

  gh api "repos/${repo}/git/refs/heads/master" -X PATCH \
    --input - <<< "$(jq -n --arg sha "$target_sha" '{sha: $sha, force: false}')" \
    || {
      echo "error: failed to update refs/heads/master." >&2
      echo "Add RELEASE_BOT_TOKEN (admin PAT or GitHub App token) to the repository ruleset bypass list." >&2
      echo "See .github/act/README.md#release-bot-setup" >&2
      exit 1
    }

  echo "Created commit ${target_sha} on master"
else
  echo "Version files already aligned on master"
fi

tag_moved=false
if [ "$tag_sha" != "$target_sha" ]; then
  if [ -n "$tag_sha" ]; then
    gh api "repos/${repo}/git/refs/${tag_ref}" -X PATCH \
      --input - <<< "$(jq -n --arg sha "$target_sha" '{sha: $sha, force: true}')"
  else
    gh api "repos/${repo}/git/refs" -X POST \
      --input - <<< "$(jq -n --arg ref "refs/tags/v${version}" --arg sha "$target_sha" '{ref: $ref, sha: $sha}')"
  fi
  tag_moved=true
  echo "Moved tag v${version} to ${target_sha}"
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "pushed_master=${needs_commit}"
    echo "tag_moved=${tag_moved}"
    echo "commit_sha=${target_sha}"
  } >> "$GITHUB_OUTPUT"
fi
