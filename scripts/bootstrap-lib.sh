#!/usr/bin/env bash
# Shared helpers for cloud bootstrap scripts.
set -euo pipefail

bootstrap_log() {
  printf '==> %s\n' "$*"
}

bootstrap_require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

bootstrap_load_env_file() {
  local file=$1
  [ -f "$file" ] || return 0
  bootstrap_log "Loading app secrets from ${file}"
  set -a
  # shellcheck disable=SC1090
  source "$file"
  set +a
}

bootstrap_set_gh_secret() {
  local repo=$1
  local name=$2
  local value=$3
  local dry_run=${4:-0}

  if [ -z "$value" ]; then
    return 0
  fi
  if [ "$dry_run" -eq 1 ]; then
    printf '  would set GitHub secret %s\n' "$name"
    return 0
  fi
  printf '%s' "$value" | gh secret set "$name" --repo "$repo"
  bootstrap_log "Set GitHub secret ${name}"
}

bootstrap_sync_common_app_secrets() {
  local repo=$1
  local dry_run=$2

  bootstrap_set_gh_secret "$repo" DATABASE_URL "${DATABASE_URL:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" KAFKA_BROKERS "${KAFKA_BROKERS:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" KAFKA_USERNAME "${KAFKA_USERNAME:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" KAFKA_PASSWORD "${KAFKA_PASSWORD:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" DISCORD_WEBHOOK_URL "${DISCORD_WEBHOOK_URL:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" SLACK_WEBHOOK_URL "${SLACK_WEBHOOK_URL:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" TELEGRAM_BOT_TOKEN "${TELEGRAM_BOT_TOKEN:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" TELEGRAM_CHAT_ID "${TELEGRAM_CHAT_ID:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" GRAFANA_CLOUD_PROMETHEUS_URL "${GRAFANA_CLOUD_PROMETHEUS_URL:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" GRAFANA_CLOUD_PROMETHEUS_USER "${GRAFANA_CLOUD_PROMETHEUS_USER:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" GRAFANA_CLOUD_API_KEY "${GRAFANA_CLOUD_API_KEY:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" GRAFANA_CLOUD_STACK_URL "${GRAFANA_CLOUD_STACK_URL:-}" "$dry_run"
  bootstrap_set_gh_secret "$repo" GRAFANA_CLOUD_PROMETHEUS_DATASOURCE_UID "${GRAFANA_CLOUD_PROMETHEUS_DATASOURCE_UID:-}" "$dry_run"
}

bootstrap_sync_routing_secrets() {
  local repo=$1
  local cloud=$2
  local target=$3
  local dry_run=$4

  bootstrap_set_gh_secret "$repo" DEPLOY_CLOUD "$cloud" "$dry_run"
  bootstrap_set_gh_secret "$repo" DEPLOY_TARGET "$target" "$dry_run"
  bootstrap_set_gh_secret "$repo" DEPLOY_METHOD terraform "$dry_run"
}

bootstrap_terraform_apply() {
  local dir=$1
  shift

  bootstrap_require_cmd terraform
  terraform -chdir="$dir" init -input=false
  if [ "$#" -gt 0 ]; then
    terraform -chdir="$dir" apply -auto-approve "$@"
  else
    terraform -chdir="$dir" apply -auto-approve
  fi
}

bootstrap_terraform_plan() {
  local dir=$1
  shift

  bootstrap_require_cmd terraform
  terraform -chdir="$dir" init -input=false
  terraform -chdir="$dir" plan "$@"
}
