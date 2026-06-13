output "vm_public_ip" {
  description = "Public IP address of the deployed VM"
  value       = length(google_compute_instance.app_vm) > 0 ? google_compute_instance.app_vm[0].network_interface[0].access_config[0].nat_ip : null
  sensitive   = true
}

output "gke_endpoint" {
  description = "GKE endpoint if deployed to Kubernetes"
  value       = length(google_container_cluster.gke) > 0 ? google_container_cluster.gke[0].endpoint : null
  sensitive   = true
}

output "cloud_run_notifier_url" {
  description = "URL of the deployed Cloud Run notification service"
  value       = length(google_cloud_run_v2_service.notification_service) > 0 ? google_cloud_run_v2_service.notification_service[0].uri : null
}
