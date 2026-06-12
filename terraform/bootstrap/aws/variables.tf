variable "region" {
  description = "AWS region for Terraform state and deployments"
  type        = string
  default     = "us-east-1"
}

variable "github_repository" {
  description = "GitHub repository in owner/name form"
  type        = string
}

variable "state_bucket_name" {
  description = "Optional globally unique S3 bucket name for Terraform state"
  type        = string
  default     = null
}

variable "deploy_role_name" {
  description = "IAM role name assumed by GitHub Actions"
  type        = string
  default     = "manga-cdc-github-deploy"
}

variable "deployment_target" {
  description = "Documented deployment target (vm, kubernetes, serverless)"
  type        = string
  default     = "serverless"

  validation {
    condition     = contains(["vm", "kubernetes", "serverless"], var.deployment_target)
    error_message = "deployment_target must be vm, kubernetes, or serverless."
  }
}
