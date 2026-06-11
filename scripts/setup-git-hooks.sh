#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

git config --local commit.gpgsign true
git config --local core.hooksPath .githooks
chmod +x .githooks/* scripts/*.sh

echo "Configured: commit.gpgsign=true, core.hooksPath=.githooks"
