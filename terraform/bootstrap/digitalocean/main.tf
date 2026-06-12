terraform {
  required_version = ">= 1.6"
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.39"
    }
  }
}

provider "digitalocean" {
  token = var.do_token
}

locals {
  state_bucket_name = coalesce(
    var.state_bucket_name,
    "mangacdc-tfstate-${substr(sha256(var.github_repository), 0, 8)}",
  )
  spaces_endpoint = "${var.spaces_region}.digitaloceanspaces.com"
}

resource "digitalocean_spaces_bucket" "tf_state" {
  name   = local.state_bucket_name
  region = var.spaces_region
  acl    = "private"
}

resource "digitalocean_spaces_key" "github_deploy" {
  name = "manga-cdc-github-deploy"
}
