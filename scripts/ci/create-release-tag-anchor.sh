#!/usr/bin/env bash
# Create an empty commit on master used as the release tag anchor.
#
# The commit message includes [skip release] [skip ci] so moving vX.Y.Z to this
# commit does not start another Release workflow.
set -euo pipefail

version="${1:?version required (e.g. 0.5.0)}"
repo="${GITHUB_REPOSITORY:?}"

if [ -z "${RELEASE_BOT_TOKEN:-}" ]; then
  echo "error: RELEASE_BOT_TOKEN is required to update protected refs" >&2
  exit 1
fi

if [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "error: GITHUB_TOKEN is required to create verified commits" >&2
  exit 1
fi

if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "Invalid semver: $version" >&2
  exit 1
fi

gh_api_content() {
  GH_TOKEN="$GITHUB_TOKEN" gh api "$@"
}

gh_api_push() {
  GH_TOKEN="$RELEASE_BOT_TOKEN" gh api "$@"
}

anchor_message="chore: release tag anchor v${version} [skip release] [skip ci]"

master_sha="$(gh_api_push "repos/${repo}/git/ref/heads/master" --jq .object.sha)"
current_message="$(gh_api_content "repos/${repo}/git/commits/${master_sha}" --jq .commit.message)"

if [ "$current_message" = "$anchor_message" ]; then
  echo "Anchor commit already at master (${master_sha})" >&2
  target_sha="$master_sha"
else
  parent_tree="$(gh_api_content "repos/${repo}/git/commits/${master_sha}" --jq .tree.sha)"
  target_sha="$(gh_api_content "repos/${repo}/git/commits" \
    --input - <<< "$(jq -n \
      --arg msg "$anchor_message" \
      --arg tree "$parent_tree" \
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
      exit 1
    }

  echo "Created release tag anchor ${target_sha} on master" >&2
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "commit_sha=${target_sha}" >> "$GITHUB_OUTPUT"
fi
