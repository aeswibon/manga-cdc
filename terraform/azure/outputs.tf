output "vm_public_ip" {
  description = "Public IP address of the deployed Azure VM"
  value       = length(azurerm_public_ip.pip) > 0 ? azurerm_public_ip.pip[0].ip_address : null
  sensitive   = true
}

output "aks_endpoint" {
  description = "AKS Kubernetes cluster API endpoint"
  value       = length(azurerm_kubernetes_cluster.aks) > 0 ? azurerm_kubernetes_cluster.aks[0].kube_config[0].host : null
  sensitive   = true
}

output "container_app_url" {
  description = "The URL of the deployed notification service Container App"
  value       = length(azurerm_container_app.notification_service) > 0 ? "https://${azurerm_container_app.notification_service[0].ingress[0].fqdn}" : null
}
