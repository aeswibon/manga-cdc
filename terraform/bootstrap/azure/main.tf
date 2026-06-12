terraform {
  required_version = ">= 1.6"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.0"
    }
  }
}

provider "azurerm" {
  features {}
}

provider "azuread" {}

data "azurerm_client_config" "current" {}

data "azurerm_subscription" "current" {}

locals {
  bootstrap_resource_group = coalesce(var.resource_group_name, "rg-manga-cdc-bootstrap")
  storage_account_name = coalesce(
    var.storage_account_name,
    substr(replace("mcdc${replace(data.azurerm_subscription.current.subscription_id, "-", "")}", "-", ""), 0, 24),
  )
  github_oidc_subjects = [
    "repo:${var.github_repository}:ref:refs/heads/master",
    "repo:${var.github_repository}:ref:refs/tags/*",
  ]
}

resource "azurerm_resource_group" "bootstrap" {
  name     = local.bootstrap_resource_group
  location = var.location
}

resource "azurerm_storage_account" "tf_state" {
  name                     = local.storage_account_name
  resource_group_name      = azurerm_resource_group.bootstrap.name
  location                 = azurerm_resource_group.bootstrap.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  min_tls_version          = "TLS1_2"
}

resource "azurerm_storage_container" "tfstate" {
  name                  = "tfstate"
  storage_account_name  = azurerm_storage_account.tf_state.name
  container_access_type = "private"
}

resource "azuread_application" "github" {
  display_name = "manga-cdc-github-deploy"
}

resource "azuread_service_principal" "github" {
  client_id = azuread_application.github.client_id
}

resource "azuread_application_federated_identity_credential" "github" {
  for_each = toset(local.github_oidc_subjects)

  application_id = azuread_application.github.id
  display_name   = replace(each.value, ":", "-")
  audiences      = ["api://AzureADTokenExchange"]
  issuer         = "https://token.actions.githubusercontent.com"
  subject        = each.value
}

resource "azurerm_role_assignment" "github_deploy" {
  scope                = data.azurerm_subscription.current.id
  role_definition_name = "Contributor"
  principal_id         = azuread_service_principal.github.object_id
}

resource "azurerm_role_assignment" "github_state" {
  scope                = azurerm_storage_account.tf_state.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azuread_service_principal.github.object_id
}
