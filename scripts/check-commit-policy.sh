#!/usr/bin/env bash
# CI guard: all commits on this branch must be GPG-signed with no Cursor co-author trailers.
set -euo pipefail

base="${1:?base ref or commit SHA required}"
if ! git rev-parse --verify "$base" >/dev/null 2>&1; then
  echo "error: invalid git ref: $base" >&2
  exit 1
fi

if git merge-base --is-ancestor "$base" HEAD 2>/dev/null && [[ "$base" != "$(git rev-parse HEAD)" ]]; then
  range="${base}..HEAD"
else
  range="HEAD"
fi

failed=0
while read -r sha; do
  status="$(git log -1 --format='%G?' "$sha")"
  if [[ "$status" != "G" ]]; then
    echo "error: commit $sha is not GPG-verified (status=$status)" >&2
    failed=1
  fi
  if git log -1 --format='%B' "$sha" | grep -qiE '^co-authored-by:[[:space:]]*cursor'; then
    echo "error: commit $sha contains a Cursor co-author trailer" >&2
    failed=1
  fi
done < <(git rev-list "$range")

exit "$failed"
