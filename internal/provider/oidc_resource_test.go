package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOidcResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "unleash_oidc" "simple_oidc_config" {
						enabled = true
						discover_url = "http://mock-openid-server:9000/.well-known/openid-configuration"
						secret = "super-secret"
						client_id = "client-id"
						auto_create = false
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_oidc.simple_oidc_config", "enabled", "true"),
					resource.TestCheckResourceAttr("unleash_oidc.simple_oidc_config", "discover_url", "http://mock-openid-server:9000/.well-known/openid-configuration"),
					resource.TestCheckResourceAttr("unleash_oidc.simple_oidc_config", "secret", "super-secret"),
					resource.TestCheckResourceAttr("unleash_oidc.simple_oidc_config", "client_id", "client-id"),
					resource.TestCheckResourceAttr("unleash_oidc.simple_oidc_config", "auto_create", "false"),
					resource.TestCheckResourceAttr("unleash_oidc.simple_oidc_config", "default_root_role", "2"), //even though we didn't set it, we expect this as a default
				),
			},
			{
				Config: `
					resource "unleash_oidc" "simple_oidc_config" {
						enabled = true
						discover_url = "http://mock-openid-server:9000/.well-known/openid-configuration"
						secret = "kinda-sorta-secret"
						client_id = "client-id"
						auto_create = true
						default_root_role = 1
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_oidc.simple_oidc_config", "secret", "kinda-sorta-secret"),
					resource.TestCheckResourceAttr("unleash_oidc.simple_oidc_config", "auto_create", "true"),
					resource.TestCheckResourceAttr("unleash_oidc.simple_oidc_config", "default_root_role", "1"),
				),
			},
		},
	})
}
