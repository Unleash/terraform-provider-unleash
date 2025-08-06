resource "unleash_saml" "simple_saml_config" {
  enabled           = true
  certificate       = "test-certificate"
  entity_id         = "some-entity-id"
  sign_on_url       = "http://other-places.com"
  auto_create       = true
  default_root_role = 1
}