terraform {
  required_version = ">= 1.6"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
  zone    = var.zone
}

data "google_client_config" "default" {
  count = var.deployment_target == "kubernetes" ? 1 : 0
}

# -----------------------------------------------------------------------------
# Database URL Parser
# -----------------------------------------------------------------------------
locals {
  # Regex to parse connection string format: postgres://user:pass@host:port/dbname?query
  db_parts          = regex("^postgres://(?:(?P<user>[^:@]+)(?::(?P<pass>[^@]+))?@)?(?P<host>[^/]+)(?P<path>/[^?]+)?(?:\\?(?P<query>.+))?$", var.database_url)
  db_user           = local.db_parts.user != null ? local.db_parts.user : ""
  db_pass           = local.db_parts.pass != null ? local.db_parts.pass : ""
  db_host           = local.db_parts.host
  db_path           = local.db_parts.path != null ? local.db_parts.path : "/postgres"
  db_query          = local.db_parts.query != null ? "?${local.db_parts.query}" : ""
  db_path_and_query = "${local.db_path}${local.db_query}"

  # Render application .env content for VM
  env_file_content = <<EOT
SCRAPER_IMAGE=${var.scraper_image}
NOTIFICATION_IMAGE=${var.notification_image}
DATABASE_URL=${var.database_url}
SPRING_DATASOURCE_URL=jdbc:postgresql://${local.db_host}${local.db_path_and_query}
SPRING_DATASOURCE_USERNAME=${local.db_user}
SPRING_DATASOURCE_PASSWORD=${local.db_pass}
KAFKA_BROKERS=${var.kafka_brokers}
KAFKA_TOPIC=mangacdc.public.chapters
KAFKA_USERNAME=${var.kafka_username}
KAFKA_PASSWORD=${var.kafka_password}
CDC_ENABLED=true
ADMIN_MUTATIONS_ENABLED=false
SECURITY_REQUIRE_API_KEY=true
SECURITY_REQUIRE_WEBHOOK_AUTH=true
API_READ_KEY=${var.api_read_key}
WEBHOOK_SECRET=${var.webhook_secret}
QSTASH_CURRENT_SIGNING_KEY=${var.qstash_current_signing_key}
QSTASH_NEXT_SIGNING_KEY=${var.qstash_next_signing_key}
ALLOWED_ORIGINS=${var.allowed_origins}
DISCORD_WEBHOOK_URL=${var.discord_webhook_url}
SLACK_WEBHOOK_URL=${var.slack_webhook_url}
TELEGRAM_BOT_TOKEN=${var.telegram_bot_token}
TELEGRAM_CHAT_ID=${var.telegram_chat_id}
OBSERVABILITY_MODE=${var.observability_mode}
GRAFANA_CLOUD_PROMETHEUS_URL=${var.grafana_cloud_prometheus_url}
GRAFANA_CLOUD_PROMETHEUS_USER=${var.grafana_cloud_prometheus_user}
GRAFANA_CLOUD_API_KEY=${var.grafana_cloud_api_key}
GRAFANA_CLOUD_STACK_URL=${var.grafana_cloud_stack_url}
EOT
}

# -----------------------------------------------------------------------------
# TARGET: VM Deployment (Docker Compose on GCE)
# -----------------------------------------------------------------------------
resource "google_service_account" "vm_sa" {
  count        = var.deployment_target == "vm" ? 1 : 0
  account_id   = "manga-cdc-vm-sa"
  display_name = "Manga CDC VM Service Account"
}

resource "google_compute_instance" "app_vm" {
  count        = var.deployment_target == "vm" ? 1 : 0
  name         = "manga-cdc-vm-${var.environment}"
  machine_type = var.machine_type
  zone         = var.zone

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
      size  = 30
    }
  }

  network_interface {
    network = "default"
    access_config {} # Ephemeral Public IP
  }

  metadata_startup_script = templatefile("${path.module}/../templates/startup.sh.tftpl", {
    env_file_content            = local.env_file_content
    compose_file_content        = file("${path.module}/../../docker-compose.prod.yml")
    caddyfile_content           = fileexists("${path.module}/../../Caddyfile") ? file("${path.module}/../../Caddyfile") : ""
    observability_cloud_enabled = var.observability_mode == "grafana-cloud"
    observability_cloud_content = file("${path.module}/../../docker-compose.observability-cloud.yml")
    alloy_config_content        = file("${path.module}/../../alloy/config.prod.alloy")
    observability_flags         = var.observability_mode == "grafana-cloud" ? "-f docker-compose.observability-cloud.yml" : ""
  })

  service_account {
    email  = google_service_account.vm_sa[0].email
    scopes = ["cloud-platform"]
  }

  labels = {
    app = "manga-cdc"
    env = var.environment
  }

  tags = ["manga-cdc-web"]
}

resource "google_compute_firewall" "allow_web" {
  count   = var.deployment_target == "vm" ? 1 : 0
  name    = "manga-cdc-allow-web-${var.environment}"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["80", "443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["manga-cdc-web"]
}

# -----------------------------------------------------------------------------
# TARGET: GKE Cluster + Helm Deployment (Kubernetes)
# -----------------------------------------------------------------------------
resource "google_container_cluster" "gke" {
  count                    = var.deployment_target == "kubernetes" ? 1 : 0
  name                     = "manga-cdc-gke-${var.environment}"
  location                 = var.region
  remove_default_node_pool = true
  initial_node_count       = 1

  release_channel {
    channel = "REGULAR"
  }
}

resource "google_container_node_pool" "gke_nodes" {
  count      = var.deployment_target == "kubernetes" ? 1 : 0
  name       = "manga-cdc-node-pool"
  location   = var.region
  cluster    = google_container_cluster.gke[0].name
  node_count = var.gke_node_count

  node_config {
    preemptible  = true
    machine_type = var.gke_node_type
    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

    labels = {
      app = "manga-cdc"
      env = var.environment
    }
  }
}

provider "kubernetes" {
  host                   = var.deployment_target == "kubernetes" ? "https://${google_container_cluster.gke[0].endpoint}" : ""
  token                  = var.deployment_target == "kubernetes" ? data.google_client_config.default[0].access_token : ""
  cluster_ca_certificate = var.deployment_target == "kubernetes" ? base64decode(google_container_cluster.gke[0].master_auth[0].cluster_ca_certificate) : ""
}

provider "helm" {
  kubernetes {
    host                   = var.deployment_target == "kubernetes" ? "https://${google_container_cluster.gke[0].endpoint}" : ""
    token                  = var.deployment_target == "kubernetes" ? data.google_client_config.default[0].access_token : ""
    cluster_ca_certificate = var.deployment_target == "kubernetes" ? base64decode(google_container_cluster.gke[0].master_auth[0].cluster_ca_certificate) : ""
  }
}

resource "helm_release" "manga_cdc" {
  count      = var.deployment_target == "kubernetes" ? 1 : 0
  name       = "manga-cdc"
  chart      = "${path.module}/../../helm/manga-cdc"
  namespace  = "default"
  depends_on = [google_container_node_pool.gke_nodes[0]]

  set {
    name  = "images.scraper"
    value = var.scraper_image
  }

  set {
    name  = "images.notification"
    value = var.notification_image
  }

  set {
    name  = "database.url"
    value = var.database_url
  }

  set {
    name  = "database.jdbcUrl"
    value = "jdbc:postgresql://${local.db_host}${local.db_path_and_query}"
  }

  set {
    name  = "database.username"
    value = local.db_user
  }

  set {
    name  = "database.password"
    value = local.db_pass
  }

  set {
    name  = "eventing.kafka.enabled"
    value = "true"
  }

  set {
    name  = "eventing.kafka.brokers"
    value = var.kafka_brokers
  }

  set {
    name  = "eventing.kafka.username"
    value = var.kafka_username
  }

  set {
    name  = "eventing.kafka.password"
    value = var.kafka_password
  }

  set {
    name  = "discord.webhookUrl"
    value = var.discord_webhook_url
  }

  set {
    name  = "slack.webhookUrl"
    value = var.slack_webhook_url
  }

  set {
    name  = "telegram.botToken"
    value = var.telegram_bot_token
  }

  set {
    name  = "telegram.chatId"
    value = var.telegram_chat_id
  }

  set {
    name  = "notifiers.discord"
    value = var.discord_webhook_url != "" ? "true" : "false"
  }

  set {
    name  = "notifiers.slack"
    value = var.slack_webhook_url != "" ? "true" : "false"
  }

  set {
    name  = "notifiers.telegram"
    value = var.telegram_bot_token != "" ? "true" : "false"
  }

  set {
    name  = "security.requireApiKey"
    value = "true"
  }

  set {
    name  = "security.requireWebhookAuth"
    value = "true"
  }

  set {
    name  = "security.adminMutationsEnabled"
    value = "false"
  }

  set {
    name  = "security.allowedOrigins"
    value = var.allowed_origins
  }

  set_sensitive {
    name  = "security.apiReadKey"
    value = var.api_read_key
  }

  set_sensitive {
    name  = "security.webhookSecret"
    value = var.webhook_secret
  }

  set_sensitive {
    name  = "security.qstashCurrentSigningKey"
    value = var.qstash_current_signing_key
  }

  set_sensitive {
    name  = "security.qstashNextSigningKey"
    value = var.qstash_next_signing_key
  }
}

# -----------------------------------------------------------------------------
# TARGET: Cloud Run + Cloud Scheduler Deployment (Serverless)
# -----------------------------------------------------------------------------
locals {
  kafka_env = var.cdc_enabled ? {
    KAFKA_BROKERS                  = var.kafka_brokers
    SPRING_KAFKA_BOOTSTRAP_SERVERS = var.kafka_brokers
    SPRING_KAFKA_SASL_MECHANISM    = "SCRAM-SHA-256"
    SPRING_KAFKA_SASL_JAAS_CONFIG  = "org.apache.kafka.common.security.scram.ScramLoginModule required username=\"${var.kafka_username}\" password=\"${var.kafka_password}\";"
    SPRING_KAFKA_SECURITY_PROTOCOL = "SASL_SSL"
    KAFKA_TOPIC                    = "mangacdc.public.chapters"
    KAFKA_USERNAME                 = var.kafka_username
    KAFKA_PASSWORD                 = var.kafka_password
    } : {
    SPRING_AUTOCONFIGURE_EXCLUDE = "org.springframework.boot.autoconfigure.kafka.KafkaAutoConfiguration"
  }

  # tomap() is required: a keyed literal with a for expression is typed as object,
  # but dynamic for_each only accepts map/set.
  cloud_run_env = tomap({
    for k, v in merge({
      DATABASE_URL                  = var.database_url
      SPRING_DATASOURCE_URL         = "jdbc:postgresql://${local.db_host}${local.db_path_and_query}"
      SPRING_DATASOURCE_USERNAME    = local.db_user
      SPRING_DATASOURCE_PASSWORD    = local.db_pass
      CDC_ENABLED                   = var.cdc_enabled ? "true" : "false"
      DB_MAX_POOL_SIZE              = "3"
      DB_MIN_IDLE                   = "0"
      ADMIN_MUTATIONS_ENABLED       = "false"
      SECURITY_REQUIRE_API_KEY      = "true"
      SECURITY_REQUIRE_WEBHOOK_AUTH = "true"
      API_READ_KEY                  = var.api_read_key
      WEBHOOK_SECRET                = var.webhook_secret
      QSTASH_CURRENT_SIGNING_KEY    = var.qstash_current_signing_key
      QSTASH_NEXT_SIGNING_KEY       = var.qstash_next_signing_key
      ALLOWED_ORIGINS               = var.allowed_origins
      DISCORD_WEBHOOK_URL           = var.discord_webhook_url
      SLACK_WEBHOOK_URL             = var.slack_webhook_url
      TELEGRAM_BOT_TOKEN            = var.telegram_bot_token
      TELEGRAM_CHAT_ID              = var.telegram_chat_id
      OBSERVABILITY_MODE            = var.observability_mode
      GRAFANA_CLOUD_PROMETHEUS_URL  = var.grafana_cloud_prometheus_url
      GRAFANA_CLOUD_PROMETHEUS_USER = var.grafana_cloud_prometheus_user
      GRAFANA_CLOUD_API_KEY         = var.grafana_cloud_api_key
      GRAFANA_CLOUD_STACK_URL       = var.grafana_cloud_stack_url
      QSTASH_TOKEN                  = var.qstash_token
    }, local.kafka_env) : k => v if v != ""
  })
}

resource "google_cloud_run_v2_job" "scraper_job" {
  count    = var.deployment_target == "serverless" ? 1 : 0
  name     = "manga-cdc-scraper-job-${var.environment}"
  location = var.region

  template {
    template {
      containers {
        image = var.scraper_image
        resources {
          limits = {
            memory = var.cloud_run_scraper_memory
            cpu    = var.cloud_run_scraper_cpu
          }
        }
        dynamic "env" {
          for_each = local.cloud_run_env
          content {
            name  = env.key
            value = env.value
          }
        }
        env {
          name  = "QSTASH_DESTINATION_URL"
          value = var.qstash_destination_url != "" ? var.qstash_destination_url : "${google_cloud_run_v2_service.notification_service[0].uri}/api/webhook"
        }
        env {
          name  = "RUN_ONCE"
          value = "true"
        }
      }
    }
  }
}

resource "google_cloud_run_v2_service" "notification_service" {
  count    = var.deployment_target == "serverless" ? 1 : 0
  name     = "manga-cdc-notifier-${var.environment}"
  location = var.region

  template {
    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }

    containers {
      image = var.notification_image
      resources {
        limits = {
          memory = var.cloud_run_notifier_memory
          cpu    = var.cloud_run_notifier_cpu
        }
        cpu_idle          = true
        startup_cpu_boost = true
      }
      ports {
        container_port = 8080
      }
      dynamic "env" {
        for_each = local.cloud_run_env
        content {
          name  = env.key
          value = env.value
        }
      }
    }
  }
}

resource "google_cloud_run_v2_service_iam_member" "noauth" {
  count    = var.deployment_target == "serverless" ? 1 : 0
  location = google_cloud_run_v2_service.notification_service[0].location
  name     = google_cloud_run_v2_service.notification_service[0].name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_service_account" "scheduler_sa" {
  count        = var.deployment_target == "serverless" ? 1 : 0
  account_id   = "manga-cdc-scheduler-sa"
  display_name = "Manga CDC Scheduler Service Account"
}

resource "google_project_iam_member" "scheduler_run_jobs" {
  count   = var.deployment_target == "serverless" ? 1 : 0
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.scheduler_sa[0].email}"
}

resource "google_cloud_scheduler_job" "scraper_trigger" {
  count            = var.deployment_target == "serverless" ? 1 : 0
  name             = "manga-cdc-scraper-trigger-${var.environment}"
  description      = "Trigger the manga-cdc scraper Cloud Run Job"
  schedule         = var.cloud_run_scheduler_schedule
  time_zone        = "Etc/UTC"
  attempt_deadline = "320s"

  retry_config {
    retry_count = 1
  }

  http_target {
    http_method = "POST"
    uri         = "https://${var.region}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/${google_cloud_run_v2_job.scraper_job[0].name}:run"

    oauth_token {
      service_account_email = google_service_account.scheduler_sa[0].email
    }
  }

  depends_on = [
    google_project_iam_member.scheduler_run_jobs
  ]
}
