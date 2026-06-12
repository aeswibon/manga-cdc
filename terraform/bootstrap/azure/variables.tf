variable "location" {
  description = "Azure region"
  type        = string
  default     = "eastus"
}

variable "github_repository" {
  description = "GitHub repository in owner/name form"
  type        = string
}

variable "resource_group_name" {
  description = "Optional resource group name for bootstrap resources"
  type        = string
  default     = null
}

variable "storage_account_name" {
  description = "Optional storage account name (3-24 lowercase alphanumeric)"
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
