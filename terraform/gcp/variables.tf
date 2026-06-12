variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "GCP zone"
  type        = string
  default     = "us-central1-a"
}

variable "environment" {
  description = "Environment name (e.g. staging, prod)"
  type        = string
  default     = "prod"
}

variable "deployment_target" {
  description = "Target runtime: 'vm' (Docker Compose on VM), 'kubernetes' (Helm on GKE), or 'serverless' (Serverless Container + Scheduler)"
  type        = string
  default     = "vm"
  validation {
    condition     = contains(["vm", "kubernetes", "serverless"], var.deployment_target)
    error_message = "deployment_target must be 'vm', 'kubernetes', or 'serverless'."
  }
}

variable "cloud_run_scheduler_schedule" {
  description = "Cron schedule for the Cloud Scheduler to trigger the scraper (e.g. every 15 mins)"
  type        = string
  default     = "*/15 * * * *"
}

variable "machine_type" {
  description = "GCP machine type for the VM"
  type        = string
  default     = "e2-medium"
}

variable "gke_node_count" {
  description = "Number of GKE worker nodes"
  type        = number
  default     = 2
}

variable "gke_node_type" {
  description = "Machine type for GKE nodes"
  type        = string
  default     = "e2-medium"
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
