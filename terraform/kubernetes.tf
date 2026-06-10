resource "digitalocean_kubernetes_cluster" "main" {
  name    = "manga-cdc-${var.environment}"
  region  = var.region
  version = "1.31"

  maintenance_policy {
    start_time = "04:00"
    day        = "sunday"
  }

  node_pool {
    name       = "worker-pool"
    size       = var.node_size
    node_count = var.node_count
    tags       = ["manga-cdc", var.environment]
  }
}
