package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "unleash_project" "test" {
						id = "default"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unleash_project.test", "name"),
					resource.TestCheckResourceAttr("data.unleash_project.test", "name", "Default"),
					resource.TestCheckResourceAttr("data.unleash_project.test", "id", "default"),
					resource.TestCheckResourceAttr("data.unleash_project.test", "description", "Default project"),
					resource.TestCheckResourceAttr("data.unleash_project.test", "mode", "open"),
				),
			},
		},
	})
}
