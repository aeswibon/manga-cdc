variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for Cloud Run and the Terraform state bucket"
  type        = string
  default     = "us-central1"
}

variable "github_repository" {
  description = "GitHub repository in owner/name form (e.g. aeswibon/manga-cdc)"
  type        = string
}

variable "state_bucket_name" {
  description = "Optional GCS bucket name for Terraform remote state (must be globally unique)"
  type        = string
  default     = null
}

variable "deploy_service_account_id" {
  description = "Service account ID (short name) used by GitHub Actions deploy workflow"
  type        = string
  default     = "manga-cdc-github-deploy"
}

variable "workload_identity_pool_id" {
  description = "Workload Identity Pool ID"
  type        = string
  default     = "github-actions"
}

variable "workload_identity_provider_id" {
  description = "Workload Identity Provider ID"
  type        = string
  default     = "github"
}

variable "deployment_target" {
  description = "Deployment target the GitHub deploy role must be able to provision (vm, kubernetes, serverless)"
  type        = string
  default     = "serverless"

  validation {
    condition     = contains(["vm", "kubernetes", "serverless"], var.deployment_target)
    error_message = "deployment_target must be vm, kubernetes, or serverless."
  }
}
