variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name (e.g. staging, prod)"
  type        = string
  default     = "prod"
}

variable "deployment_target" {
  description = "Target runtime: 'vm' (Docker Compose on EC2), 'kubernetes' (Helm on EKS), or 'serverless' (Serverless Container on Fargate + EventBridge)"
  type        = string
  default     = "vm"
  validation {
    condition     = contains(["vm", "kubernetes", "serverless"], var.deployment_target)
    error_message = "deployment_target must be 'vm', 'kubernetes', or 'serverless'."
  }
}

variable "aws_scheduler_schedule" {
  description = "Cron schedule for EventBridge to trigger the Fargate scraper task (e.g. every 15 mins)"
  type        = string
  default     = "rate(15 minutes)"
}

variable "instance_type" {
  description = "EC2 Instance type for VM deployment"
  type        = string
  default     = "t3.medium"
}

variable "eks_node_count" {
  description = "Number of worker nodes in EKS node group"
  type        = number
  default     = 2
}

variable "eks_node_type" {
  description = "Instance type for EKS worker nodes"
  type        = string
  default     = "t3.medium"
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
