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

set +e
terraform plan -refresh=false -input=false -lock=false -detailed-exitcode \
  "${plan_vars[@]}"
plan_exit=$?
set -e

# 0 = no changes, 2 = changes present — both mean the configuration planned successfully.
if [ "$plan_exit" -eq 0 ] || [ "$plan_exit" -eq 2 ]; then
  echo "terraform plan succeeded for ${cloud}/${target} (exit ${plan_exit})"
  exit 0
fi

echo "terraform plan failed for ${cloud}/${target} (exit ${plan_exit})" >&2
exit "$plan_exit"
