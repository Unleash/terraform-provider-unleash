package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSamlResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "unleash_saml" "simple_saml_config" {
						enabled = true
						certificate = "test-certificate"
						entity_id = "some-entity-id"
						sign_on_url = "http://places.com"
						auto_create = false
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_saml.simple_saml_config", "enabled", "true"),
					resource.TestCheckResourceAttr("unleash_saml.simple_saml_config", "certificate", "test-certificate"),
					resource.TestCheckResourceAttr("unleash_saml.simple_saml_config", "entity_id", "some-entity-id"),
					resource.TestCheckResourceAttr("unleash_saml.simple_saml_config", "sign_on_url", "http://places.com"),
					resource.TestCheckResourceAttr("unleash_saml.simple_saml_config", "auto_create", "false"),
					resource.TestCheckResourceAttr("unleash_saml.simple_saml_config", "default_root_role", "2"), //even though we didn't set it, we expect this as a default
				),
			},
			{
				Config: `
					resource "unleash_saml" "simple_saml_config" {
						enabled = true
						certificate = "test-certificate"
						entity_id = "some-entity-id"
						sign_on_url = "http://other-places.com"
						auto_create = true
						default_root_role = 1
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_saml.simple_saml_config", "sign_on_url", "http://other-places.com"),
					resource.TestCheckResourceAttr("unleash_saml.simple_saml_config", "auto_create", "true"),
					resource.TestCheckResourceAttr("unleash_saml.simple_saml_config", "default_root_role", "1"),
				),
			},
		},
	})
}
