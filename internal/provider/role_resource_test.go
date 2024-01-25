package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// customCheckRolePermissionExists checks if a specific permission (and optionally an environment) exists in a role's permissions list.
func customCheckRolePermissionExists(resourceName string, permissionName string, environment ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		for i := 0; ; i++ {
			permissionAttrKey := fmt.Sprintf("permissions.%d.name", i)
			permission, ok := rs.Primary.Attributes[permissionAttrKey]
			if !ok {
				break // Exit the loop if no more permissions are found
			}
			if permission == permissionName {
				// Check for environment if provided
				if len(environment) > 0 {
					environmentAttrKey := fmt.Sprintf("permissions.%d.environment", i)
					env, ok := rs.Primary.Attributes[environmentAttrKey]
					if !ok || env != environment[0] {
						continue // Skip to the next permission if environment does not match
					}
				}
				return nil
			}
		}

		return fmt.Errorf("Permission %s with environment %v not found in resource %s", permissionName, environment, resourceName)
	}
}

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
					customCheckRolePermissionExists("unleash_role.custom_root_role", "CREATE_PROJECT"),
					customCheckRolePermissionExists("unleash_role.custom_root_role", "UPDATE_PROJECT"),
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
					customCheckRolePermissionExists("unleash_role.custom_root_role", "CREATE_SEGMENT"),
					customCheckRolePermissionExists("unleash_role.custom_root_role", "UPDATE_SEGMENT"),
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
					customCheckRolePermissionExists("unleash_role.project_role", "CREATE_FEATURE"),
					customCheckRolePermissionExists("unleash_role.project_role", "DELETE_FEATURE"),
					customCheckRolePermissionExists("unleash_role.project_role", "UPDATE_FEATURE_ENVIRONMENT", "development"),
				),
			},
			{
				Config:            `resource "unleash_role" "project_role" {}`,
				ResourceName:      "unleash_role.project_role",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
