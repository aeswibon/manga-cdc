variable "ci_plan_mode" {
  description = "When true, use offline-friendly Azure provider settings for CI terraform plan."
  type        = bool
  default     = false
}

variable "location" {
  description = "Azure region location"
  type        = string
  default     = "East US"
}

variable "environment" {
  description = "Environment name (e.g. staging, prod)"
  type        = string
  default     = "prod"
}

variable "deployment_target" {
  description = "Target runtime: 'vm' (Docker Compose on Azure VM), 'kubernetes' (Helm on AKS), or 'serverless' (Serverless Azure Container Apps + Job)"
  type        = string
  default     = "vm"
  validation {
    condition     = contains(["vm", "kubernetes", "serverless"], var.deployment_target)
    error_message = "deployment_target must be 'vm', 'kubernetes', or 'serverless'."
  }
}

variable "azure_job_schedule" {
  description = "Cron schedule for the Container App Job to trigger the scraper (e.g. every 15 mins)"
  type        = string
  default     = "*/15 * * * *"
}

variable "vm_size" {
  description = "Azure VM size for VM deployment"
  type        = string
  default     = "Standard_B2s"
}

variable "ssh_public_key" {
  description = "SSH public key content for VM access"
  type        = string
  default     = ""
}

variable "aks_node_count" {
  description = "Number of worker nodes in AKS node pool"
  type        = number
  default     = 2
}

variable "aks_node_size" {
  description = "VM size for AKS worker nodes"
  type        = string
  default     = "Standard_B2s"
}

# Application Config & Secrets
variable "scraper_image" {
  description = "Docker image for the scraper service"
  type        = string
}

variable "notification_image" {
  description = "Docker image for the notification service"
  type        = string
}

variable "database_url" {
  description = "PostgreSQL connection URL"
  type        = string
  sensitive   = true
}

variable "kafka_brokers" {
  description = "Kafka bootstrap brokers comma-separated list"
  type        = string
}

variable "kafka_username" {
  description = "Kafka SCRAM username"
  type        = string
}

variable "kafka_password" {
  description = "Kafka SCRAM password"
  type        = string
  sensitive   = true
}

variable "discord_webhook_url" {
  description = "Discord Webhook URL for notifications"
  type        = string
  default     = ""
}

variable "slack_webhook_url" {
  description = "Slack Webhook URL for notifications"
  type        = string
  default     = ""
}

variable "telegram_bot_token" {
  description = "Telegram Bot Token for notifications"
  type        = string
  default     = ""
}

variable "telegram_chat_id" {
  description = "Telegram Chat ID for notifications"
  type        = string
  default     = ""
}

# Observability
variable "observability_mode" {
  description = "Observability mode: grafana-cloud, self-hosted, or off"
  type        = string
  default     = "grafana-cloud"
}

variable "grafana_cloud_prometheus_url" {
  description = "Grafana Cloud Prometheus push URL"
  type        = string
  default     = ""
}

variable "grafana_cloud_prometheus_user" {
  description = "Grafana Cloud Prometheus instance ID"
  type        = string
  default     = ""
}

variable "grafana_cloud_api_key" {
  description = "Grafana Cloud Access Token with metrics:write"
  type        = string
  default     = ""
  sensitive   = true
}

variable "grafana_cloud_stack_url" {
  description = "Grafana Cloud stack URL (e.g. https://yourstack.grafana.net)"
  type        = string
  default     = ""
}

variable "qstash_token" {
  description = "Upstash QStash Token for serverless event delivery"
  type        = string
  default     = ""
  sensitive   = true
}

variable "qstash_destination_url" {
  description = "Target URL for QStash webhooks (notification service endpoint)"
  type        = string
  default     = ""
}

variable "api_read_key" {
  description = "Shared read API key for notifier read endpoints and actuator metrics"
  type        = string
  default     = ""
  sensitive   = true
}

variable "webhook_secret" {
  description = "Fallback shared secret for POST /api/webhook when QStash signing is unavailable"
  type        = string
  default     = ""
  sensitive   = true
}

variable "qstash_current_signing_key" {
  description = "Upstash QStash current signing key for webhook verification"
  type        = string
  default     = ""
  sensitive   = true
}

variable "qstash_next_signing_key" {
  description = "Upstash QStash next signing key for webhook verification"
  type        = string
  default     = ""
  sensitive   = true
}

variable "allowed_origins" {
  description = "Comma-separated browser origins allowed to call the notifier API"
  type        = string
  default     = ""
}
