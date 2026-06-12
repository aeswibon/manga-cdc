#!/usr/bin/env bash
# Cloud-agnostic one-time bootstrap for CI/CD deploy prerequisites:
# - Remote Terraform state storage
# - GitHub Actions cloud authentication (OIDC / federated identity where supported)
# - Optional GitHub repository secret sync via gh CLI
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=scripts/bootstrap-lib.sh
source "${ROOT}/scripts/bootstrap-lib.sh"

CLOUD=""
TARGET="serverless"
DRY_RUN=0
SKIP_GH_SECRETS=0
ENV_FILE="${ROOT}/.env"
GITHUB_REPO=""
PROJECT_ID=""
REGION=""
LOCATION=""
SPACES_REGION="nyc3"
DO_TOKEN=""

usage() {
  cat <<'EOF'
Usage: bootstrap.sh --cloud <gcp|aws|azure|digitalocean> [options]

Options:
  --cloud NAME          Cloud provider (required)
  --target TARGET       Deployment target: vm, kubernetes, serverless (default: serverless)
  --project-id ID       GCP project ID (default: gcloud config project)
  --region REGION       Region (GCP/AWS default: us-central1 / us-east-1)
  --location REGION     Azure location (default: eastus)
  --spaces-region REG   DigitalOcean Spaces region (default: nyc3)
  --do-token TOKEN      DigitalOcean API token (default: DIGITALOCEAN_ACCESS_TOKEN env)
  --repo OWNER/NAME     GitHub repo (default: gh repo view)
  --env-file PATH       Load app secrets from file (default: .env if present)
  --skip-gh-secrets     Run Terraform only; skip gh secret set
  --dry-run             Terraform plan only
  -h, --help            Show this help

Prerequisites vary by cloud:
  gcp          gcloud auth application-default login
  aws          aws configure / AWS credentials in environment
  azure        az login
  digitalocean DIGITALOCEAN_ACCESS_TOKEN or --do-token

Examples:
  ./scripts/bootstrap.sh --cloud gcp --target serverless
  ./scripts/bootstrap.sh --cloud aws --target serverless --region us-east-1
  ./scripts/bootstrap.sh --cloud azure --target vm
  ./scripts/bootstrap.sh --cloud digitalocean --target serverless --spaces-region sfo3
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --cloud)
      CLOUD="${2:?}"
      shift 2
      ;;
    --target)
      TARGET="${2:?}"
      shift 2
      ;;
    --project-id)
      PROJECT_ID="${2:?}"
      shift 2
      ;;
    --region)
      REGION="${2:?}"
      shift 2
      ;;
    --location)
      LOCATION="${2:?}"
      shift 2
      ;;
    --spaces-region)
      SPACES_REGION="${2:?}"
      shift 2
      ;;
    --do-token)
      DO_TOKEN="${2:?}"
      shift 2
      ;;
    --repo)
      GITHUB_REPO="${2:?}"
      shift 2
      ;;
    --env-file)
      ENV_FILE="${2:?}"
      shift 2
      ;;
    --skip-gh-secrets)
      SKIP_GH_SECRETS=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ -z "$CLOUD" ]; then
  echo "error: --cloud is required" >&2
  usage >&2
  exit 1
fi

case "$CLOUD" in
  gcp | aws | azure | digitalocean) ;;
  *)
    echo "error: unsupported cloud: $CLOUD" >&2
    exit 1
    ;;
esac

case "$TARGET" in
  vm | kubernetes | serverless) ;;
  *)
    echo "error: unsupported target: $TARGET" >&2
    exit 1
    ;;
esac

if [ -z "$GITHUB_REPO" ]; then
  bootstrap_require_cmd gh
  GITHUB_REPO="$(gh repo view --json nameWithOwner -q .nameWithOwner)"
fi

bootstrap_load_env_file "$ENV_FILE"

BOOTSTRAP_DIR="${ROOT}/terraform/bootstrap/${CLOUD}"
if [ ! -d "$BOOTSTRAP_DIR" ]; then
  echo "error: bootstrap module not found: ${BOOTSTRAP_DIR}" >&2
  exit 1
fi

bootstrap_log "Cloud:  ${CLOUD}"
bootstrap_log "Target: ${TARGET}"
bootstrap_log "Repo:   ${GITHUB_REPO}"

sync_cloud_secrets() {
  local repo=$1
  local dry_run=$2

  bootstrap_sync_routing_secrets "$repo" "$CLOUD" "$TARGET" "$dry_run"
  bootstrap_sync_common_app_secrets "$repo" "$dry_run"

  case "$CLOUD" in
    gcp)
      bootstrap_set_gh_secret "$repo" GCP_PROJECT_ID "$PROJECT_ID" "$dry_run"
      bootstrap_set_gh_secret "$repo" GCP_REGION "$REGION" "$dry_run"
      bootstrap_set_gh_secret "$repo" TF_STATE_BUCKET "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw tf_state_bucket)" "$dry_run"
      bootstrap_set_gh_secret "$repo" GCP_WORKLOAD_IDENTITY_PROVIDER "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw gcp_workload_identity_provider)" "$dry_run"
      bootstrap_set_gh_secret "$repo" GCP_SERVICE_ACCOUNT "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw gcp_service_account)" "$dry_run"
      ;;
    aws)
      bootstrap_set_gh_secret "$repo" AWS_REGION "$REGION" "$dry_run"
      bootstrap_set_gh_secret "$repo" TF_STATE_BUCKET "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw tf_state_bucket)" "$dry_run"
      bootstrap_set_gh_secret "$repo" AWS_ROLE_ARN "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw aws_role_arn)" "$dry_run"
      ;;
    azure)
      bootstrap_set_gh_secret "$repo" AZURE_CLIENT_ID "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw azure_client_id)" "$dry_run"
      bootstrap_set_gh_secret "$repo" AZURE_TENANT_ID "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw azure_tenant_id)" "$dry_run"
      bootstrap_set_gh_secret "$repo" AZURE_SUBSCRIPTION_ID "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw azure_subscription_id)" "$dry_run"
      bootstrap_set_gh_secret "$repo" AZURE_RESOURCE_GROUP "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw azure_resource_group)" "$dry_run"
      bootstrap_set_gh_secret "$repo" AZURE_STORAGE_ACCOUNT "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw azure_storage_account)" "$dry_run"
      ;;
    digitalocean)
      bootstrap_set_gh_secret "$repo" DIGITALOCEAN_ACCESS_TOKEN "$DO_TOKEN" "$dry_run"
      bootstrap_set_gh_secret "$repo" TF_STATE_BUCKET "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw tf_state_bucket)" "$dry_run"
      bootstrap_set_gh_secret "$repo" DO_SPACE_ENDPOINT "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw do_space_endpoint)" "$dry_run"
      bootstrap_set_gh_secret "$repo" DO_SPACES_ACCESS_KEY "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw do_spaces_access_key)" "$dry_run"
      bootstrap_set_gh_secret "$repo" DO_SPACES_SECRET_KEY "$(terraform -chdir="$BOOTSTRAP_DIR" output -raw do_spaces_secret_key)" "$dry_run"
      ;;
  esac
}

case "$CLOUD" in
  gcp)
    bootstrap_require_cmd gcloud
    if [ -z "$PROJECT_ID" ]; then
      PROJECT_ID="$(gcloud config get-value project 2>/dev/null || true)"
    fi
    if [ -z "$PROJECT_ID" ] || [ "$PROJECT_ID" = "(unset)" ]; then
      echo "error: GCP project not set; pass --project-id or run gcloud config set project" >&2
      exit 1
    fi
    REGION="${REGION:-us-central1}"
    TF_ARGS=(
      -var="project_id=${PROJECT_ID}"
      -var="region=${REGION}"
      -var="github_repository=${GITHUB_REPO}"
      -var="deployment_target=${TARGET}"
    )
    ;;
  aws)
    bootstrap_require_cmd aws
    REGION="${REGION:-us-east-1}"
    TF_ARGS=(
      -var="region=${REGION}"
      -var="github_repository=${GITHUB_REPO}"
      -var="deployment_target=${TARGET}"
    )
    ;;
  azure)
    bootstrap_require_cmd az
    LOCATION="${LOCATION:-eastus}"
    TF_ARGS=(
      -var="location=${LOCATION}"
      -var="github_repository=${GITHUB_REPO}"
      -var="deployment_target=${TARGET}"
    )
    ;;
  digitalocean)
    DO_TOKEN="${DO_TOKEN:-${DIGITALOCEAN_ACCESS_TOKEN:-}}"
    if [ -z "$DO_TOKEN" ]; then
      echo "error: DigitalOcean token required; set DIGITALOCEAN_ACCESS_TOKEN or pass --do-token" >&2
      exit 1
    fi
    export TF_VAR_do_token="$DO_TOKEN"
    TF_ARGS=(
      -var="spaces_region=${SPACES_REGION}"
      -var="github_repository=${GITHUB_REPO}"
      -var="deployment_target=${TARGET}"
    )
    ;;
esac

if [ "$DRY_RUN" -eq 1 ]; then
  bootstrap_log "Dry run — Terraform plan only"
  bootstrap_terraform_plan "$BOOTSTRAP_DIR" "${TF_ARGS[@]}"
  exit 0
fi

bootstrap_terraform_apply "$BOOTSTRAP_DIR" "${TF_ARGS[@]}"

cat <<EOF

Bootstrap complete for ${CLOUD} (${TARGET}).

Next:
  1. Push a release tag (v*) so GHCR images exist.
  2. Deploy with DEPLOY_METHOD=terraform (already set if gh sync ran).
  3. After the first successful deploy, set DEPLOY_METHOD=direct for image-only tag updates.

EOF

if [ "$SKIP_GH_SECRETS" -eq 1 ]; then
  bootstrap_log "Skipping GitHub secret sync (--skip-gh-secrets)"
  exit 0
fi

bootstrap_require_cmd gh
bootstrap_log "Syncing GitHub repository secrets"
sync_cloud_secrets "$GITHUB_REPO" 0
bootstrap_log "Done."
