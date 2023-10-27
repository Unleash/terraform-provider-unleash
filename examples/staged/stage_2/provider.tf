terraform {
  required_providers {
    unleash = {
      source  = "Unleash/unleash"
      version = "1.0.0"
    }
  }
}

provider "unleash" {
  base_url      = "http://localhost:4242"
  authorization = "*:*.unleash-insecure-admin-api-token"
}
