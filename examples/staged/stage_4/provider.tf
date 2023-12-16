terraform {
  required_providers {
    unleash = {
      source  = "Unleash/unleash"
      version = "~> 1"
    }
  }
}

provider "unleash" {
  base_url      = "http://localhost:4242"
  authorization = "*:*.unleash-insecure-admin-api-token"
}
