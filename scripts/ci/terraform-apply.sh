#!/usr/bin/env bash
# Run terraform apply for a cloud module. Expects secrets via environment variables.
set -euo pipefail

cloud="${1:?cloud provider required (gcp|aws|azure|digitalocean)}"
: "${DEPLOYMENT_TARGET:?}"
: "${SCRAPER_IMAGE:?}"
: "${NOTIFICATION_IMAGE:?}"
: "${IMAGE_TAG:?}"
: "${DATABASE_URL:?}"

cd "terraform/${cloud}"

if [ "$cloud" = "gcp" ] && [ -n "${TF_STATE_BUCKET:-}" ]; then
  cat > backend.tf <<EOF
terraform {
  backend "gcs" {
    bucket = "${TF_STATE_BUCKET}"
    prefix = "terraform/state"
  }
}
EOF
elif [ "$cloud" = "aws" ] && [ -n "${TF_STATE_BUCKET:-}" ]; then
  cat > backend.tf <<EOF
terraform {
  backend "s3" {
    bucket = "${TF_STATE_BUCKET}"
    key    = "terraform/state/terraform.tfstate"
    region = "${AWS_REGION:-us-east-1}"
  }
}
EOF
elif [ "$cloud" = "azure" ] && [ -n "${AZURE_STORAGE_ACCOUNT:-}" ]; then
  cat > backend.tf <<EOF
terraform {
  backend "azurerm" {
    resource_group_name  = "${AZURE_RESOURCE_GROUP}"
    storage_account_name = "${AZURE_STORAGE_ACCOUNT}"
    container_name       = "tfstate"
    key                  = "terraform.tfstate"
  }
}
EOF
elif [ "$cloud" = "digitalocean" ] && [ -n "${TF_STATE_BUCKET:-}" ] && [ -n "${DO_SPACE_ENDPOINT:-}" ]; then
  cat > backend.tf <<EOF
terraform {
  backend "s3" {
    endpoint                    = "${DO_SPACE_ENDPOINT}"
    region                      = "us-east-1"
    bucket                      = "${TF_STATE_BUCKET}"
    key                         = "terraform.tfstate"
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
  }
}
EOF
fi

provider_vars=()
case "$cloud" in
  digitalocean)
    provider_vars+=(-var="do_token=${DIGITALOCEAN_ACCESS_TOKEN:-}")
    ;;
  gcp)
    provider_vars+=(-var="project_id=${GCP_PROJECT_ID:-}")
    provider_vars+=(-var="region=${GCP_REGION:-us-central1}")
    ;;
esac

terraform init
terraform apply -auto-approve \
  "${provider_vars[@]}" \
  -var="deployment_target=${DEPLOYMENT_TARGET}" \
  -var="scraper_image=${SCRAPER_IMAGE}:${IMAGE_TAG}" \
  -var="notification_image=${NOTIFICATION_IMAGE}:${IMAGE_TAG}" \
  -var="database_url=${DATABASE_URL}" \
  -var="kafka_brokers=${KAFKA_BROKERS:-}" \
  -var="kafka_username=${KAFKA_USERNAME:-}" \
  -var="kafka_password=${KAFKA_PASSWORD:-}" \
  -var="discord_webhook_url=${DISCORD_WEBHOOK_URL:-}" \
  -var="slack_webhook_url=${SLACK_WEBHOOK_URL:-}" \
  -var="telegram_bot_token=${TELEGRAM_BOT_TOKEN:-}" \
  -var="telegram_chat_id=${TELEGRAM_CHAT_ID:-}" \
  -var="observability_mode=grafana-cloud" \
  -var="grafana_cloud_prometheus_url=${GRAFANA_CLOUD_PROMETHEUS_URL:-}" \
  -var="grafana_cloud_prometheus_user=${GRAFANA_CLOUD_PROMETHEUS_USER:-}" \
  -var="grafana_cloud_api_key=${GRAFANA_CLOUD_API_KEY:-}" \
  -var="grafana_cloud_stack_url=${GRAFANA_CLOUD_STACK_URL:-}" \
  -var="qstash_token=${QSTASH_TOKEN:-}" \
  -var="qstash_destination_url=${QSTASH_DESTINATION_URL:-}"
