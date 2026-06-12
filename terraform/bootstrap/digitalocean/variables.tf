variable "do_token" {
  description = "DigitalOcean API token with read/write access"
  type        = string
  sensitive   = true
}

variable "spaces_region" {
  description = "DigitalOcean Spaces region slug (e.g. nyc3, sfo3)"
  type        = string
  default     = "nyc3"
}

variable "github_repository" {
  description = "GitHub repository in owner/name form"
  type        = string
}

variable "state_bucket_name" {
  description = "Optional Spaces bucket name for Terraform state"
  type        = string
  default     = null
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
