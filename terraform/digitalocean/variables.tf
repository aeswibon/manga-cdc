variable "do_token" {
  description = "DigitalOcean API token"
  type        = string
  sensitive   = true
}

variable "region" {
  description = "DigitalOcean region"
  type        = string
  default     = "sfo3"
}

variable "environment" {
  description = "Environment name (e.g. staging, prod)"
  type        = string
  default     = "prod"
}

variable "deployment_target" {
  description = "Target runtime: 'vm' (Docker Compose on Droplet), 'kubernetes' (Helm on DOKS), or 'serverless' (DigitalOcean App Platform Service/Worker)"
  type        = string
  default     = "vm"
  validation {
    condition     = contains(["vm", "kubernetes", "serverless"], var.deployment_target)
    error_message = "deployment_target must be 'vm', 'kubernetes', or 'serverless'."
  }
}

variable "droplet_size" {
  description = "Size of the droplet for VM deployment"
  type        = string
  default     = "s-2vcpu-2gb"
}

variable "doks_node_count" {
  description = "Number of Kubernetes worker nodes"
  type        = number
  default     = 2
}

variable "doks_node_size" {
  description = "Size of Kubernetes worker nodes"
  type        = string
  default     = "s-2vcpu-2gb"
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
