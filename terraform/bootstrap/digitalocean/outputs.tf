output "spaces_region" {
  value = var.spaces_region
}

output "tf_state_bucket" {
  value = digitalocean_spaces_bucket.tf_state.name
}

output "do_space_endpoint" {
  value = local.spaces_endpoint
}

output "do_spaces_access_key" {
  value     = digitalocean_spaces_key.github_deploy.access_key
  sensitive = true
}

output "do_spaces_secret_key" {
  value     = digitalocean_spaces_key.github_deploy.secret_key
  sensitive = true
}
