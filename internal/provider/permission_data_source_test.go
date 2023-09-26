package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPermissionDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "unleash_permission" "create_project" {
						name = "CREATE_PROJECT"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.unleash_permission.create_project", "id", "12"),
					resource.TestCheckResourceAttr("data.unleash_permission.create_project", "name", "CREATE_PROJECT"),
					resource.TestCheckResourceAttr("data.unleash_permission.create_project", "type", "root"),
				),
			},
			{
				Config: `
					data "unleash_permission" "update_project" {
						name = "UPDATE_PROJECT"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.unleash_permission.update_project", "id", "13"),
					resource.TestCheckResourceAttr("data.unleash_permission.update_project", "name", "UPDATE_PROJECT"),
					resource.TestCheckResourceAttr("data.unleash_permission.update_project", "type", "project"),
				),
			},
			{
				Config: `
					data "unleash_permission" "create_feature_strategy" {
						name = "CREATE_FEATURE_STRATEGY"
						environment = "development"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.unleash_permission.create_feature_strategy", "id", "25"),
					resource.TestCheckResourceAttr("data.unleash_permission.create_feature_strategy", "name", "CREATE_FEATURE_STRATEGY"),
					resource.TestCheckResourceAttr("data.unleash_permission.create_feature_strategy", "type", "environment"),
					resource.TestCheckResourceAttr("data.unleash_permission.create_feature_strategy", "environment", "development"),
				),
			},
		},
	})
}
