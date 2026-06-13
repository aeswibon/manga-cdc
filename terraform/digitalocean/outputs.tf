output "droplet_public_ip" {
  description = "Public IP address of the deployed Droplet"
  value       = length(digitalocean_droplet.app_droplet) > 0 ? digitalocean_droplet.app_droplet[0].ipv4_address : null
  sensitive   = true
}

output "doks_endpoint" {
  description = "DOKS API endpoint if deployed to Kubernetes"
  value       = length(digitalocean_kubernetes_cluster.doks) > 0 ? digitalocean_kubernetes_cluster.doks[0].endpoint : null
  sensitive   = true
}

output "app_platform_url" {
  description = "The live URL of the deployed App Platform app"
  value       = length(digitalocean_app.manga_cdc) > 0 ? digitalocean_app.manga_cdc[0].live_url : null
}
