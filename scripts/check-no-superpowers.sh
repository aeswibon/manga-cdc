#!/usr/bin/env bash
set -euo pipefail

tracked="$(git ls-files 'docs/superpowers' '.superpowers' 2>/dev/null || true)"
if [[ -n "$tracked" ]]; then
  echo "error: docs/superpowers and .superpowers must not be tracked in git:" >&2
  echo "$tracked" >&2
  exit 1
fi

staged="$(git diff --cached --name-only -- 'docs/superpowers' '.superpowers' 2>/dev/null || true)"
if [[ -n "$staged" ]]; then
  echo "error: docs/superpowers and .superpowers must not be staged for commit:" >&2
  echo "$staged" >&2
  exit 1
fi
