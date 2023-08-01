package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "unleash_user" "the_newbie" {
						username = "test"
						name = "Test User"
						email = "test@getunleash.io"
						root_role = "2"
						send_email = false
					}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify placeholder id attribute
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "id", "2"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "username", "test"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "name", "Test User"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "email", "test@getunleash.io"),
					//resource.TestCheckResourceAttr("unleash_user.the_newbie", "root_role", "2"),
				),
			},
		},
	})
}
