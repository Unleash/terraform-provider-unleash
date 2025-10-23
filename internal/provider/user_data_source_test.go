package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "unleash_user" "admin_user" {
						id = 1
					}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify placeholder id attribute
					resource.TestCheckResourceAttr("data.unleash_user.admin_user", "id", "1"),
					resource.TestCheckResourceAttr("data.unleash_user.admin_user", "username", "admin"),
					resource.TestCheckResourceAttr("data.unleash_user.admin_user", "root_role", "1"),
					// by default, the admin user does not get a name or email
					resource.TestCheckNoResourceAttr("data.unleash_user.admin_user", "name"),
					resource.TestCheckNoResourceAttr("data.unleash_user.admin_user", "email"),
				),
			},
			{
				Config: `
					resource "unleash_user" "searchable" {
						username   = "tf-acc-search-user"
						email      = "tf-acc-search-user@getunleash.io"
						name       = "Terraform Search User"
						root_role  = 2
						send_email = false
					}

					data "unleash_user" "search_by_email" {
						email = unleash_user.searchable.email
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unleash_user.search_by_email", "id"),
					resource.TestCheckResourceAttr("data.unleash_user.search_by_email", "email", "tf-acc-search-user@getunleash.io"),
					resource.TestCheckResourceAttr("data.unleash_user.search_by_email", "username", "tf-acc-search-user"),
					resource.TestCheckResourceAttr("data.unleash_user.search_by_email", "root_role", "2"),
					resource.TestCheckResourceAttr("data.unleash_user.search_by_email", "name", "Terraform Search User"),
				),
			},
		},
	})
}
