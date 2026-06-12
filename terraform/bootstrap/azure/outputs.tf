output "location" {
  value = var.location
}

output "azure_resource_group" {
  value = azurerm_resource_group.bootstrap.name
}

output "azure_storage_account" {
  value = azurerm_storage_account.tf_state.name
}

output "tf_state_container" {
  value = azurerm_storage_container.tfstate.name
}

output "azure_client_id" {
  value = azuread_application.github.client_id
}

output "azure_tenant_id" {
  value = data.azurerm_client_config.current.tenant_id
}

output "azure_subscription_id" {
  value = data.azurerm_subscription.current.subscription_id
}
