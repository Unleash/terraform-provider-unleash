package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccContextFieldDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "unleash_context_field" "built_in_context_field" {
						name = "appName"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.unleash_context_field.built_in_context_field", "name", "appName"),
					resource.TestCheckResourceAttr("data.unleash_context_field.built_in_context_field", "description", "Allows you to constrain on application name"),
				),
			},
		},
	})
}
