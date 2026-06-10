output "cluster_id" {
  value       = digitalocean_kubernetes_cluster.main.id
  description = "The ID of the Kubernetes cluster"
}

output "cluster_name" {
  value       = digitalocean_kubernetes_cluster.main.name
  description = "The name of the Kubernetes cluster"
}

output "cluster_endpoint" {
  value       = digitalocean_kubernetes_cluster.main.endpoint
  description = "The API endpoint of the Kubernetes cluster"
  sensitive   = true
}

output "kube_config" {
  value       = digitalocean_kubernetes_cluster.main.kube_config[0].raw_config
  description = "The raw kubeconfig for the cluster"
  sensitive   = true
}

output "registry_url" {
  value       = digitalocean_container_registry.main.server_url
  description = "The container registry URL"
}

output "postgres_host" {
  value       = var.enable_managed_db ? digitalocean_database_cluster.postgres[0].host : null
  description = "The hostname of the managed PostgreSQL database"
  sensitive   = true
}

output "postgres_port" {
  value       = var.enable_managed_db ? digitalocean_database_cluster.postgres[0].port : null
  description = "The port of the managed PostgreSQL database"
}
