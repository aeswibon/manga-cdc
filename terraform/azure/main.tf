terraform {
  required_version = ">= 1.6"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
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

provider "azurerm" {
  features {}

  subscription_id            = var.ci_plan_mode ? "11111111-1111-1111-1111-111111111111" : null
  tenant_id                  = var.ci_plan_mode ? "22222222-2222-2222-2222-222222222222" : null
  skip_provider_registration = var.ci_plan_mode
}

resource "azurerm_resource_group" "rg" {
  name     = "rg-manga-cdc-${var.environment}"
  location = var.location
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

  # Render application .env content for Azure VM
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
# TARGET: VM Deployment (Docker Compose on Azure Linux VM)
# -----------------------------------------------------------------------------
resource "azurerm_virtual_network" "vnet" {
  count               = var.deployment_target == "vm" ? 1 : 0
  name                = "manga-cdc-vnet-${var.environment}"
  address_space       = ["10.0.0.0/16"]
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
}

resource "azurerm_subnet" "subnet" {
  count                = var.deployment_target == "vm" ? 1 : 0
  name                 = "manga-cdc-subnet"
  resource_group_name  = azurerm_resource_group.rg.name
  virtual_network_name = azurerm_virtual_network.vnet[0].name
  address_prefixes     = ["10.0.1.0/24"]
}

resource "azurerm_public_ip" "pip" {
  count               = var.deployment_target == "vm" ? 1 : 0
  name                = "manga-cdc-pip"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  allocation_method   = "Dynamic"
}

resource "azurerm_network_security_group" "nsg" {
  count               = var.deployment_target == "vm" ? 1 : 0
  name                = "manga-cdc-nsg"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name

  security_rule {
    name                       = "SSH"
    priority                   = 1001
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "HTTP"
    priority                   = 1002
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "HTTPS"
    priority                   = 1003
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}

resource "azurerm_network_interface" "nic" {
  count               = var.deployment_target == "vm" ? 1 : 0
  name                = "manga-cdc-nic"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.subnet[0].id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.pip[0].id
  }
}

resource "azurerm_network_interface_security_group_association" "nic_nsg" {
  count                     = var.deployment_target == "vm" ? 1 : 0
  network_interface_id      = azurerm_network_interface.nic[0].id
  network_security_group_id = azurerm_network_security_group.nsg[0].id
}

resource "azurerm_linux_virtual_machine" "app_vm" {
  count               = var.deployment_target == "vm" ? 1 : 0
  name                = "manga-cdc-vm-${var.environment}"
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  size                = var.vm_size
  admin_username      = "azureuser"

  network_interface_ids = [
    azurerm_network_interface.nic[0].id,
  ]

  admin_ssh_key {
    username   = "azureuser"
    public_key = var.ssh_public_key != "" ? var.ssh_public_key : "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tB placeholder"
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts"
    version   = "latest"
  }

  custom_data = base64encode(templatefile("${path.module}/../templates/startup.sh.tftpl", {
    env_file_content            = local.env_file_content
    compose_file_content        = file("${path.module}/../../docker-compose.prod.yml")
    caddyfile_content           = fileexists("${path.module}/../../Caddyfile") ? file("${path.module}/../../Caddyfile") : ""
    observability_cloud_enabled = var.observability_mode == "grafana-cloud"
    observability_cloud_content = file("${path.module}/../../docker-compose.observability-cloud.yml")
    alloy_config_content        = file("${path.module}/../../alloy/config.prod.alloy")
    observability_flags         = var.observability_mode == "grafana-cloud" ? "-f docker-compose.observability-cloud.yml" : ""
  }))
}

# -----------------------------------------------------------------------------
# TARGET: AKS Cluster + Helm Deployment (Kubernetes)
# -----------------------------------------------------------------------------
resource "azurerm_kubernetes_cluster" "aks" {
  count               = var.deployment_target == "kubernetes" ? 1 : 0
  name                = "manga-cdc-aks-${var.environment}"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  dns_prefix          = "mangacdc"

  default_node_pool {
    name       = "default"
    node_count = var.aks_node_count
    vm_size    = var.aks_node_size
  }

  identity {
    type = "SystemAssigned"
  }

  tags = {
    Environment = var.environment
  }
}

provider "kubernetes" {
  host                   = var.deployment_target == "kubernetes" ? azurerm_kubernetes_cluster.aks[0].kube_config[0].host : ""
  client_certificate     = var.deployment_target == "kubernetes" ? base64decode(azurerm_kubernetes_cluster.aks[0].kube_config[0].client_certificate) : ""
  client_key             = var.deployment_target == "kubernetes" ? base64decode(azurerm_kubernetes_cluster.aks[0].kube_config[0].client_key) : ""
  cluster_ca_certificate = var.deployment_target == "kubernetes" ? base64decode(azurerm_kubernetes_cluster.aks[0].kube_config[0].cluster_ca_certificate) : ""
}

provider "helm" {
  kubernetes {
    host                   = var.deployment_target == "kubernetes" ? azurerm_kubernetes_cluster.aks[0].kube_config[0].host : ""
    client_certificate     = var.deployment_target == "kubernetes" ? base64decode(azurerm_kubernetes_cluster.aks[0].kube_config[0].client_certificate) : ""
    client_key             = var.deployment_target == "kubernetes" ? base64decode(azurerm_kubernetes_cluster.aks[0].kube_config[0].client_key) : ""
    cluster_ca_certificate = var.deployment_target == "kubernetes" ? base64decode(azurerm_kubernetes_cluster.aks[0].kube_config[0].cluster_ca_certificate) : ""
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
# TARGET: Azure Container Apps + Jobs (Serverless)
# -----------------------------------------------------------------------------
resource "azurerm_log_analytics_workspace" "workspace" {
  count               = var.deployment_target == "serverless" ? 1 : 0
  name                = "manga-cdc-law-${var.environment}"
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

resource "azurerm_container_app_environment" "env" {
  count                      = var.deployment_target == "serverless" ? 1 : 0
  name                       = "manga-cdc-env-${var.environment}"
  location                   = azurerm_resource_group.rg.location
  resource_group_name        = azurerm_resource_group.rg.name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.workspace[0].id
}

locals {
  azure_env_all = {
    DATABASE_URL                  = var.database_url
    SPRING_DATASOURCE_URL         = "jdbc:postgresql://${local.db_host}${local.db_path_and_query}"
    SPRING_DATASOURCE_USERNAME    = local.db_user
    SPRING_DATASOURCE_PASSWORD    = local.db_pass
    KAFKA_BROKERS                 = var.kafka_brokers
    SPRING_KAFKA_BOOTSTRAP_SERVERS = var.kafka_brokers
    KAFKA_TOPIC                   = "mangacdc.public.chapters"
    KAFKA_USERNAME                = var.kafka_username
    KAFKA_PASSWORD                = var.kafka_password
    CDC_ENABLED                   = "true"
    DISCORD_WEBHOOK_URL           = var.discord_webhook_url
    SLACK_WEBHOOK_URL             = var.slack_webhook_url
    TELEGRAM_BOT_TOKEN            = var.telegram_bot_token
    TELEGRAM_CHAT_ID              = var.telegram_chat_id
    OBSERVABILITY_MODE            = var.observability_mode
    GRAFANA_CLOUD_PROMETHEUS_URL  = var.grafana_cloud_prometheus_url
    GRAFANA_CLOUD_PROMETHEUS_USER = var.grafana_cloud_prometheus_user
    GRAFANA_CLOUD_API_KEY         = var.grafana_cloud_api_key
    GRAFANA_CLOUD_STACK_URL       = var.grafana_cloud_stack_url
  }

  sensitive_keys = [
    "DATABASE_URL",
    "SPRING_DATASOURCE_PASSWORD",
    "KAFKA_PASSWORD",
    "GRAFANA_CLOUD_API_KEY",
    "TELEGRAM_BOT_TOKEN",
    "DISCORD_WEBHOOK_URL",
    "SLACK_WEBHOOK_URL"
  ]

  # Filter non-empty env vars
  azure_env_filtered = {
    for k, v in local.azure_env_all : k => v if v != ""
  }

  # Split into secret and general maps
  azure_secrets = {
    for k, v in local.azure_env_filtered : k => v if contains(local.sensitive_keys, k)
  }

  azure_general = {
    for k, v in local.azure_env_filtered : k => v if !contains(local.sensitive_keys, k)
  }
}

resource "azurerm_container_app" "notification_service" {
  count                        = var.deployment_target == "serverless" ? 1 : 0
  name                         = "manga-cdc-notifier-${var.environment}"
  container_app_environment_id = azurerm_container_app_environment.env[0].id
  resource_group_name          = azurerm_resource_group.rg.name
  revision_mode                = "Single"

  dynamic "secret" {
    for_each = local.azure_secrets
    content {
      name  = lower(replace(secret.key, "_", "-"))
      value = secret.value
    }
  }

  template {
    container {
      name   = "notifier"
      image  = var.notification_image
      cpu    = "0.25"
      memory = "0.5Gi"

      dynamic "env" {
        for_each = local.azure_general
        content {
          name  = env.key
          value = env.value
        }
      }

      dynamic "env" {
        for_each = local.azure_secrets
        content {
          name        = env.key
          secret_name = lower(replace(env.key, "_", "-"))
        }
      }
    }
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = true
    target_port                = 8080
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }
}

resource "azurerm_container_app_job" "scraper_job" {
  count                        = var.deployment_target == "serverless" ? 1 : 0
  name                         = "manga-cdc-scraper-job-${var.environment}"
  location                     = azurerm_resource_group.rg.location
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.env[0].id

  replica_timeout_in_seconds = 600
  replica_retry_limit        = 1

  schedule_trigger_config {
    cron_expression          = var.azure_job_schedule
    parallelism              = 1
    replica_completion_count = 1
  }

  dynamic "secret" {
    for_each = local.azure_secrets
    content {
      name  = lower(replace(secret.key, "_", "-"))
      value = secret.value
    }
  }

  template {
    container {
      name   = "scraper"
      image  = var.scraper_image
      cpu    = "0.25"
      memory = "0.5Gi"

      dynamic "env" {
        for_each = local.azure_general
        content {
          name  = env.key
          value = env.value
        }
      }

      dynamic "env" {
        for_each = local.azure_secrets
        content {
          name        = env.key
          secret_name = lower(replace(env.key, "_", "-"))
        }
      }

      env {
        name  = "RUN_ONCE"
        value = "true"
      }
    }
  }
}


