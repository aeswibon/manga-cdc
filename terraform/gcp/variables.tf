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
  description = "Cron schedule for the Cloud Scheduler to trigger the scraper (every 6 hours)"
  type        = string
  default     = "0 */6 * * *"
}

variable "cdc_enabled" {
  description = "Enable Kafka CDC consumption on the notifier (keep false on serverless to reduce Cloud Run cost)"
  type        = bool
  default     = false
}

variable "cloud_run_notifier_memory" {
  description = "Memory limit for the notifier Cloud Run service"
  type        = string
  default     = "512Mi"
}

variable "cloud_run_notifier_cpu" {
  description = "CPU limit for the notifier Cloud Run service"
  type        = string
  default     = "1"
}

variable "cloud_run_scraper_memory" {
  description = "Memory limit for the scraper Cloud Run job"
  type        = string
  default     = "512Mi"
}

variable "cloud_run_scraper_cpu" {
  description = "CPU limit for the scraper Cloud Run job"
  type        = string
  default     = "1"
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
  description = "Observability mode: grafana-cloud, self-hosted, or off (off reduces metrics export cost on serverless)"
  type        = string
  default     = "off"
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

variable "watchlist_url" {
  description = "HTTPS URL for the community watchlist YAML (raw GitHub URL in production)"
  type        = string
  default     = "https://raw.githubusercontent.com/aeswibon/manga-cdc/master/data/watchlist.yaml"
}
