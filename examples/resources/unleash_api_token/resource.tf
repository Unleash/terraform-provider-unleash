resource "unleash_api_token" "client_token" {
  token_name  = "client_token"
  type        = "client"
  expires_at  = "2024-12-31T23:59:59Z"
  project     = "default"
  environment = "development"
}