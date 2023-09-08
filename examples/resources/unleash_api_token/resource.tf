resource "unleash_api_token" "client_token" {
  token_name  = "client_token"
  type        = "client"
  expires_at  = "2024-12-31T23:59:59Z"
  project     = "default"
  environment = "development"
}

resource "unleash_api_token" "frontend_token" {
  token_name  = "frontend_token"
  type        = "frontend"
  expires_at  = "2024-12-31T23:59:59Z"
  projects    = ["*"]
  environment = "development"
}

resource "unleash_api_token" "admin_token" {
  token_name  = "admin_token"
  type        = "admin"
  expires_at  = "2024-12-31T23:59:59Z"
  projects    = ["*"]
  environment = "*"
}

resource "unleash_api_token" "admin_no_expire" {
  token_name = "admin_no_expire"
  type       = "admin"
}