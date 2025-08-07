resource "unleash_oidc" "simple_oidc_config" {
  enabled           = true
  discover_url      = "http://mock-openid-server:9000/.well-known/openid-configuration"
  secret            = "kinda-sorta-secret"
  client_id         = "client-id"
  auto_create       = true
  default_root_role = 1
}