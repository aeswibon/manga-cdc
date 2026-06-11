#!/usr/bin/env bash
# Rewrites the current commit message (drops Cursor co-author trailers) and re-signs.
set -euo pipefail

tmp="$(mktemp)"
git log -1 --format=%B | awk 'tolower($0) !~ /^co-authored-by: cursor/ { print }' >"$tmp"
# trim trailing blank lines
sed -i '' -e :a -e '/^\n*$/{$d;N;ba' -e '}' "$tmp" 2>/dev/null || sed -i -e :a -e '/^\n*$/{$d;N;ba' -e '}' "$tmp"
git commit --amend -S -F "$tmp"
rm -f "$tmp"
