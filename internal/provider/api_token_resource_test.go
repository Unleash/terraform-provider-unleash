package provider

import (
	"testing"
	"os"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApiTokenResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "unleash_api_token" "frontend_token" {
					token_name = "frontend_token"
					type = "frontend"
					expires_at = "2024-12-31T23:59:59Z"
					projects = ["*"]
					environment = "development"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.frontend_token", "secret"),
					resource.TestCheckResourceAttrSet("unleash_api_token.frontend_token", "expires_at"),
					resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "token_name", "frontend_token"),
					resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "environment", "development"),
					resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "projects.0", "*"),
				),
			},
			{
				Config: `
				resource "unleash_api_token" "client_token" {
					token_name = "client_token"
					type = "client"
					expires_at = "2024-12-31T23:59:59Z"
					projects = ["default"]
					environment = "development"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.client_token", "secret"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "expires_at", "2024-12-31T23:59:59Z"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "token_name", "client_token"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "environment", "development"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "projects.0", "default"),
				),
			},
			{ // test change expire date for previous token
				Config: `
				resource "unleash_api_token" "client_token" {
					token_name = "client_token"
					type = "client"
					expires_at = "2025-01-01T12:00:00Z"
					projects = ["default"]
					environment = "development"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.client_token", "secret"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "expires_at", "2025-01-01T12:00:00Z"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "token_name", "client_token"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "environment", "development"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "projects.0", "default"),
				),
			},
			{
				Config: `
				resource "unleash_api_token" "client_no_expire" {
					token_name = "client_no_expire"
					type = "client"
					projects = ["default"]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.client_no_expire", "secret"),
					resource.TestCheckNoResourceAttr("unleash_api_token.client_no_expire", "expires_at"),
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "token_name", "client_no_expire"),
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "environment", func() string { if v := os.Getenv("DEFAULT_ENVIRONMENT"); v != "" { return v } else { return "development" } }()),
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "projects.0", "default"),
				),
			},
		},
	})
}
