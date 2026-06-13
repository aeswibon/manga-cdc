terraform {
  required_version = ">= 1.6"
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.39"
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

provider "digitalocean" {
  token = var.do_token
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

  # Render application .env content for Droplet
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
# TARGET: VM Deployment (Docker Compose on DO Droplet)
# -----------------------------------------------------------------------------
resource "digitalocean_droplet" "app_droplet" {
  count  = var.deployment_target == "vm" ? 1 : 0
  image  = "ubuntu-22-04-x64"
  name   = "manga-cdc-droplet-${var.environment}"
  region = var.region
  size   = var.droplet_size
  tags   = ["manga-cdc", var.environment]

  user_data = templatefile("${path.module}/../templates/startup.sh.tftpl", {
    env_file_content            = local.env_file_content
    compose_file_content        = file("${path.module}/../../docker-compose.prod.yml")
    caddyfile_content           = fileexists("${path.module}/../../Caddyfile") ? file("${path.module}/../../Caddyfile") : ""
    observability_cloud_enabled = var.observability_mode == "grafana-cloud"
    observability_cloud_content = file("${path.module}/../../docker-compose.observability-cloud.yml")
    alloy_config_content        = file("${path.module}/../../alloy/config.prod.alloy")
    observability_flags         = var.observability_mode == "grafana-cloud" ? "-f docker-compose.observability-cloud.yml" : ""
  })
}

resource "digitalocean_firewall" "vm_fw" {
  count = var.deployment_target == "vm" ? 1 : 0
  name  = "manga-cdc-vm-fw-${var.environment}"

  droplet_ids = [digitalocean_droplet.app_droplet[0].id]

  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }

  outbound_rule {
    protocol              = "icmp"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}

# -----------------------------------------------------------------------------
# TARGET: DO Kubernetes (DOKS) + Helm Deployment
# -----------------------------------------------------------------------------
resource "digitalocean_kubernetes_cluster" "doks" {
  count   = var.deployment_target == "kubernetes" ? 1 : 0
  name    = "manga-cdc-doks-${var.environment}"
  region  = var.region
  version = "1.31"

  node_pool {
    name       = "worker-pool"
    size       = var.doks_node_size
    node_count = var.doks_node_count
    tags       = ["manga-cdc", var.environment]
  }
}

provider "kubernetes" {
  host                   = var.deployment_target == "kubernetes" ? digitalocean_kubernetes_cluster.doks[0].endpoint : ""
  token                  = var.deployment_target == "kubernetes" ? digitalocean_kubernetes_cluster.doks[0].kube_config[0].token : ""
  cluster_ca_certificate = var.deployment_target == "kubernetes" ? base64decode(digitalocean_kubernetes_cluster.doks[0].kube_config[0].cluster_ca_certificate) : ""
}

provider "helm" {
  kubernetes {
    host                   = var.deployment_target == "kubernetes" ? digitalocean_kubernetes_cluster.doks[0].endpoint : ""
    token                  = var.deployment_target == "kubernetes" ? digitalocean_kubernetes_cluster.doks[0].kube_config[0].token : ""
    cluster_ca_certificate = var.deployment_target == "kubernetes" ? base64decode(digitalocean_kubernetes_cluster.doks[0].kube_config[0].cluster_ca_certificate) : ""
  }
}

resource "helm_release" "manga_cdc" {
  count     = var.deployment_target == "kubernetes" ? 1 : 0
  name      = "manga-cdc"
  chart     = "${path.module}/../../helm/manga-cdc"
  namespace = "default"

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
}

# -----------------------------------------------------------------------------
# TARGET: DigitalOcean App Platform (Serverless)
# -----------------------------------------------------------------------------
locals {
  # Parse scraper image
  scraper_parts      = split(":", var.scraper_image)
  scraper_tag        = length(local.scraper_parts) > 1 ? local.scraper_parts[1] : "latest"
  scraper_path_parts = split("/", local.scraper_parts[0])
  scraper_registry   = local.scraper_path_parts[1]
  scraper_repo       = join("/", slice(local.scraper_path_parts, 2, length(local.scraper_path_parts)))

  # Parse notification image
  notifier_parts      = split(":", var.notification_image)
  notifier_tag        = length(local.notifier_parts) > 1 ? local.notifier_parts[1] : "latest"
  notifier_path_parts = split("/", local.notifier_parts[0])
  notifier_registry   = local.notifier_path_parts[1]
  notifier_repo       = join("/", slice(local.notifier_path_parts, 2, length(local.notifier_path_parts)))

  do_app_env = tomap({
    for k, v in {
      DATABASE_URL                              = var.database_url
      SPRING_DATASOURCE_URL                     = "jdbc:postgresql://${local.db_host}${local.db_path_and_query}"
      SPRING_DATASOURCE_USERNAME                = local.db_user
      SPRING_DATASOURCE_PASSWORD                = local.db_pass
      KAFKA_BROKERS                             = var.kafka_brokers
      SPRING_KAFKA_BOOTSTRAP_SERVERS            = var.kafka_brokers
      SPRING_KAFKA_PROPERTIES_SASL_MECHANISM    = "SCRAM-SHA-256"
      SPRING_KAFKA_PROPERTIES_SASL_JAAS_CONFIG  = "org.apache.kafka.common.security.scram.ScramLoginModule required username=\"${var.kafka_username}\" password=\"${var.kafka_password}\";"
      SPRING_KAFKA_PROPERTIES_SECURITY_PROTOCOL = "SASL_SSL"
      KAFKA_TOPIC                               = "mangacdc.public.chapters"
      KAFKA_USERNAME                            = var.kafka_username
      KAFKA_PASSWORD                            = var.kafka_password
      CDC_ENABLED                               = "true"
      DISCORD_WEBHOOK_URL                       = var.discord_webhook_url
      SLACK_WEBHOOK_URL                         = var.slack_webhook_url
      TELEGRAM_BOT_TOKEN                        = var.telegram_bot_token
      TELEGRAM_CHAT_ID                          = var.telegram_chat_id
      OBSERVABILITY_MODE                        = var.observability_mode
      GRAFANA_CLOUD_PROMETHEUS_URL              = var.grafana_cloud_prometheus_url
      GRAFANA_CLOUD_PROMETHEUS_USER             = var.grafana_cloud_prometheus_user
      GRAFANA_CLOUD_API_KEY                     = var.grafana_cloud_api_key
      GRAFANA_CLOUD_STACK_URL                   = var.grafana_cloud_stack_url
    } : k => v if v != ""
  })
}

resource "digitalocean_app" "manga_cdc" {
  count = var.deployment_target == "serverless" ? 1 : 0
  spec {
    name   = "manga-cdc-${var.environment}"
    region = var.region

    # Notifier web service
    service {
      name               = "notifier"
      instance_count     = 1
      instance_size_slug = "apps-s-1vcpu-0.5gb"
      http_port          = 8080

      image {
        registry_type = "GHCR"
        registry      = local.notifier_registry
        repository    = local.notifier_repo
        tag           = local.notifier_tag
      }

      dynamic "env" {
        for_each = local.do_app_env
        content {
          key   = env.key
          value = env.value
          type  = contains(["DATABASE_URL", "KAFKA_PASSWORD", "GRAFANA_CLOUD_API_KEY", "TELEGRAM_BOT_TOKEN", "DISCORD_WEBHOOK_URL", "SLACK_WEBHOOK_URL"], env.key) ? "SECRET" : "GENERAL"
        }
      }
    }

    # Scraper background worker (continuous loop)
    worker {
      name               = "scraper"
      instance_count     = 1
      instance_size_slug = "apps-s-1vcpu-0.5gb"

      image {
        registry_type = "GHCR"
        registry      = local.scraper_registry
        repository    = local.scraper_repo
        tag           = local.scraper_tag
      }

      dynamic "env" {
        for_each = local.do_app_env
        content {
          key   = env.key
          value = env.value
          type  = contains(["DATABASE_URL", "KAFKA_PASSWORD", "GRAFANA_CLOUD_API_KEY", "TELEGRAM_BOT_TOKEN", "DISCORD_WEBHOOK_URL", "SLACK_WEBHOOK_URL"], env.key) ? "SECRET" : "GENERAL"
        }
      }
    }
  }
}

