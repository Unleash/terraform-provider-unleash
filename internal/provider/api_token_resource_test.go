package provider

import (
	"testing"

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
					resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "project", "*"),
					resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "projects.0", "*"),
				),
			},
			{
				Config: `
				resource "unleash_api_token" "client_token" {
					token_name = "client_token"
					type = "client"
					expires_at = "2024-12-31T23:59:59Z"
					project = "default"
					environment = "development"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.client_token", "secret"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "expires_at", "2024-12-31T23:59:59Z"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "token_name", "client_token"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "environment", "development"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "project", "default"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "projects.0", "default"),
				),
			},
			{ // test change expire date for previous token
				Config: `
				resource "unleash_api_token" "client_token" {
					token_name = "client_token"
					type = "client"
					expires_at = "2025-01-01T12:00:00Z"
					project = "default"
					environment = "development"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.client_token", "secret"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "expires_at", "2025-01-01T12:00:00Z"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "token_name", "client_token"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "environment", "development"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "project", "default"),
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
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "environment", "default"),
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "project", "default"),
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "projects.0", "default"),
				),
			},
			{
				Config: `
				resource "unleash_api_token" "admin_token" {
					token_name = "admin_token"
					type = "admin"
					expires_at = "2024-12-31T23:59:59Z"
					projects = ["*"]
					environment = "*"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.admin_token", "secret"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_token", "expires_at", "2024-12-31T23:59:59Z"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_token", "token_name", "admin_token"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_token", "environment", "*"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_token", "project", "*"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_token", "projects.0", "*"),
				),
			},
			{
				Config: `
				resource "unleash_api_token" "admin_no_expire" {
					token_name = "admin_no_expire"
					type = "admin"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.admin_no_expire", "secret"),
					resource.TestCheckNoResourceAttr("unleash_api_token.admin_no_expire", "expires_at"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_no_expire", "token_name", "admin_no_expire"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_no_expire", "environment", "*"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_no_expire", "project", "*"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_no_expire", "projects.0", "*"),
				),
			},
			{ // test add expire date to previous token
				Config: `
				resource "unleash_api_token" "admin_no_expire" {
					token_name = "admin_no_expire"
					type = "admin"
					expires_at = "2024-12-31T23:59:59Z"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.admin_no_expire", "secret"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_no_expire", "expires_at", "2024-12-31T23:59:59Z"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_no_expire", "token_name", "admin_no_expire"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_no_expire", "environment", "*"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_no_expire", "project", "*"),
					resource.TestCheckResourceAttr("unleash_api_token.admin_no_expire", "projects.0", "*"),
				),
			},
		},
	})
}
