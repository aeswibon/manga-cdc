output "project_id" {
  description = "GCP project ID"
  value       = var.project_id
}

output "region" {
  description = "GCP region used for serverless resources"
  value       = var.region
}

output "tf_state_bucket" {
  description = "GCS bucket for manga-cdc Terraform remote state"
  value       = google_storage_bucket.tf_state.name
}

output "gcp_workload_identity_provider" {
  description = "Full Workload Identity Provider resource name for GitHub Actions"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "gcp_service_account" {
  description = "GitHub Actions deploy service account email"
  value       = google_service_account.github_deploy.email
}

output "github_repository" {
  description = "GitHub repository bound to Workload Identity Federation"
  value       = var.github_repository
}
