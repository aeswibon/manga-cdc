#!/usr/bin/env bash
# Update the dashboard Cloud Run service image on GCP (direct deploy path).
set -euo pipefail

: "${DASHBOARD_IMAGE:?DASHBOARD_IMAGE is required}"
: "${IMAGE_TAG:?IMAGE_TAG is required}"
: "${GCP_REGION:?GCP_REGION is required}"

service="${GCP_DASHBOARD_CLOUD_RUN_SERVICE:-}"
if [ -z "$service" ]; then
  echo "GCP_DASHBOARD_CLOUD_RUN_SERVICE not set; skipping dashboard Cloud Run deploy."
  exit 0
fi

image="${DASHBOARD_IMAGE}:${IMAGE_TAG}"
echo "Updating Cloud Run service ${service} in ${GCP_REGION} to ${image}"
gcloud run services update "$service" \
  --image="$image" \
  --region="$GCP_REGION" \
  --quiet
