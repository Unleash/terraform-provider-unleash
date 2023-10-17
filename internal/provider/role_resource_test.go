package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRoleResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "unleash_role" "custom_root_role" {
					name = "A custom role"
					type = "root-custom"
					description = "A custom test root role"
					permissions = [{
						name = "CREATE_PROJECT"
					}, {
						name = "UPDATE_PROJECT"
					}]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_role.custom_root_role", "id"),
					resource.TestCheckResourceAttr("unleash_role.custom_root_role", "name", "A custom role"),
					resource.TestCheckResourceAttr("unleash_role.custom_root_role", "type", "root-custom"),
					resource.TestCheckResourceAttr("unleash_role.custom_root_role", "permissions.0.name", "CREATE_PROJECT"),
					resource.TestCheckResourceAttr("unleash_role.custom_root_role", "permissions.1.name", "UPDATE_PROJECT"),
				),
			},
			// Test update name and permissions
			{
				Config: `
				resource "unleash_role" "custom_root_role" {
					name = "Renamed custom role"
					type = "root-custom"
					description = "A custom test root role"
					permissions = [{
						name = "CREATE_SEGMENT"
					}, {
						name = "UPDATE_SEGMENT"
					}]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_role.custom_root_role", "id"),
					resource.TestCheckResourceAttr("unleash_role.custom_root_role", "name", "Renamed custom role"),
					resource.TestCheckResourceAttr("unleash_role.custom_root_role", "type", "root-custom"),
					resource.TestCheckResourceAttr("unleash_role.custom_root_role", "permissions.0.name", "CREATE_SEGMENT"),
					resource.TestCheckResourceAttr("unleash_role.custom_root_role", "permissions.1.name", "UPDATE_SEGMENT"),
				),
			},
			{
				Config: `
				resource "unleash_role" "project_role" {
					name = "Custom project role"
					description = "A custom test project role"
					type = "custom"
					permissions = [{
						name = "CREATE_FEATURE"
					}, {
						name = "DELETE_FEATURE"
					}, {

						name = "UPDATE_FEATURE_ENVIRONMENT"
						environment = "development"
					}]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_role.project_role", "id"),
					resource.TestCheckResourceAttr("unleash_role.project_role", "name", "Custom project role"),
					resource.TestCheckResourceAttr("unleash_role.project_role", "type", "custom"),
					resource.TestCheckResourceAttr("unleash_role.project_role", "permissions.0.name", "CREATE_FEATURE"),
					resource.TestCheckResourceAttr("unleash_role.project_role", "permissions.1.name", "DELETE_FEATURE"),
					resource.TestCheckResourceAttr("unleash_role.project_role", "permissions.2.name", "UPDATE_FEATURE_ENVIRONMENT"),
					resource.TestCheckResourceAttr("unleash_role.project_role", "permissions.2.environment", "development"),
				),
			},
		},
	})
}
