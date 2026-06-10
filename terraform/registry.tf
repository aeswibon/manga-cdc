resource "digitalocean_container_registry" "main" {
  name                   = "manga-cdc"
  subscription_tier_slug = "basic"
  region                 = var.region
}
