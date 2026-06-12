#!/usr/bin/env bash
# Validate and plan a cloud module for a specific deployment_target using CI dummy inputs.
set -euo pipefail

cloud="${1:?cloud provider required (gcp|aws|azure|digitalocean)}"
target="${2:?deployment_target required (vm|kubernetes|serverless)}"

case "$cloud" in
  gcp | aws | azure | digitalocean) ;;
  *)
    echo "error: unsupported cloud: $cloud" >&2
    exit 1
    ;;
esac

case "$target" in
  vm | kubernetes | serverless) ;;
  *)
    echo "error: unsupported deployment_target: $target" >&2
    exit 1
    ;;
esac

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
module_dir="${repo_root}/terraform/${cloud}"

common_vars=(
  -var="deployment_target=${target}"
  -var="scraper_image=ghcr.io/example/manga-cdc/scraper:ci"
  -var="notification_image=ghcr.io/example/manga-cdc/notification-service:ci"
  -var="database_url=postgres://ci:ci@localhost:5432/mangacdc?sslmode=disable"
  -var="kafka_brokers=localhost:9092"
  -var="kafka_username=ci"
  -var="kafka_password=ci"
)

extra_vars=()
case "$cloud" in
  gcp)
    extra_vars+=(-var="project_id=ci-validate-project")
    export GOOGLE_OAUTH_ACCESS_TOKEN="${GOOGLE_OAUTH_ACCESS_TOKEN:-ci-dummy-token}"
    export GOOGLE_PROJECT="${GOOGLE_PROJECT:-ci-validate-project}"
    ;;
  aws)
    extra_vars+=(-var="ci_plan_mode=true")
    export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-testing}"
    export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-testing}"
    export AWS_EC2_METADATA_DISABLED="${AWS_EC2_METADATA_DISABLED:-true}"
    ;;
  azure)
    extra_vars+=(-var="ci_plan_mode=true")
    export ARM_USE_CLI="${ARM_USE_CLI:-false}"
    export ARM_SKIP_PROVIDER_REGISTRATION="${ARM_SKIP_PROVIDER_REGISTRATION:-true}"
    export ARM_SUBSCRIPTION_ID="${ARM_SUBSCRIPTION_ID:-11111111-1111-1111-1111-111111111111}"
    export ARM_TENANT_ID="${ARM_TENANT_ID:-22222222-2222-2222-2222-222222222222}"
    export ARM_CLIENT_ID="${ARM_CLIENT_ID:-33333333-3333-3333-3333-333333333333}"
    export ARM_CLIENT_SECRET="${ARM_CLIENT_SECRET:-ci-dummy-secret}"
    ;;
  digitalocean)
    extra_vars+=(-var="do_token=ci-dummy-token")
    export DIGITALOCEAN_TOKEN="${DIGITALOCEAN_TOKEN:-ci-dummy-token}"
    ;;
esac

plan_vars=( "${common_vars[@]}" )
if ((${#extra_vars[@]} > 0)); then
  plan_vars+=( "${extra_vars[@]}" )
fi

cd "$module_dir"

terraform init -input=false -backend=false
terraform validate

if [ "$cloud" = "azure" ]; then
  # azurerm still requires a live Azure token to plan; validate configuration offline instead.
  echo "terraform validate succeeded for ${cloud}/${target} (Azure plan skipped in CI offline mode)"
  exit 0
fi

plan_out=$(mktemp)
set +e
terraform plan -no-color -refresh=false -input=false -lock=false -detailed-exitcode \
  "${plan_vars[@]}" | tee "$plan_out"
plan_exit=$?
set -e

# 0 = no changes, 2 = changes present — both mean the configuration planned successfully.
if [ "$plan_exit" -eq 0 ] || [ "$plan_exit" -eq 2 ]; then
  echo "terraform plan succeeded for ${cloud}/${target} (exit ${plan_exit})"
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    {
      echo "### Terraform Plan: \`${cloud}/${target}\`"
      echo '```'
      cat "$plan_out"
      echo '```'
    } >> "$GITHUB_STEP_SUMMARY"
  fi
  rm -f "$plan_out"
  exit 0
fi

echo "terraform plan failed for ${cloud}/${target} (exit ${plan_exit})" >&2
rm -f "$plan_out"
exit "$plan_exit"
