resource "unleash_api_token" "frontend_token" {
  token_name  = "frontend_token"
  type        = "frontend"
  expires_at  = "2024-12-31T23:59:59Z"
  projects    = ["*"]
  environment = "development"
}
