resource "digitalocean_database_cluster" "postgres" {
  count      = var.enable_managed_db ? 1 : 0
  name       = "manga-cdc-${var.environment}"
  engine     = "pg"
  version    = "16"
  size       = "db-s-2vcpu-2gb"
  region     = var.region
  node_count = 1
}

resource "digitalocean_database_db" "main" {
  count      = var.enable_managed_db ? 1 : 0
  cluster_id = digitalocean_database_cluster.postgres[0].id
  name       = "mangacdc"
}

resource "digitalocean_database_user" "app" {
  count      = var.enable_managed_db ? 1 : 0
  cluster_id = digitalocean_database_cluster.postgres[0].id
  name       = "mangacdc"
}

resource "digitalocean_database_firewall" "main" {
  count      = var.enable_managed_db ? 1 : 0
  cluster_id = digitalocean_database_cluster.postgres[0].id

  rule {
    type  = "k8s"
    value = digitalocean_kubernetes_cluster.main.id
  }
}
