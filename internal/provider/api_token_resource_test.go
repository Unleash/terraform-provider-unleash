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
			// {
			// 	Config: `
			// 	resource "unleash_api_token" "frontend_token" {
			// 		token_name = "frontend_token"
			// 		type = "frontend"
			// 		expires_at = "2024-12-31T23:59:59Z"
			// 		projects = ["*"]
			// 		environment = "development"
			// 	}`,
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttrSet("unleash_api_token.frontend_token", "secret"),
			// 		resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "token_name", "frontend_token"),
			// 		resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "environment", "development"),
			// 		resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "project", "*"),
			// 		resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "projects.0", "*"),
			// 	),
			// },
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
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "token_name", "client_token"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "environment", "development"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "project", "default"),
					// TODO resource.TestCheckResourceAttr("unleash_api_token.client_token", "projects.0", "default"),
				),
			},
			// {
			// 	Config: testAccSampleProjectResource("RenamedToThisString", "TestId"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttrSet("unleash_project.test_project", "id"),
			// 		resource.TestCheckResourceAttrSet("unleash_project.test_project", "name"),
			// 		resource.TestCheckResourceAttr("unleash_project.test_project", "name", "RenamedToThisString"),
			// 		resource.TestCheckResourceAttr("unleash_project.test_project", "description", "test description"),
			// 	),
			// },
			// {
			// 	Config: testAccSampleProjectResourceWithNoDescription("NoDescription", "TestId2"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttrSet("unleash_project.test_project2", "id"),
			// 		resource.TestCheckResourceAttrSet("unleash_project.test_project2", "name"),
			// 	),
			// },
		},
	})
}
