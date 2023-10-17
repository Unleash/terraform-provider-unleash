package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRoleDataSource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "unleash_role" "admin_role" {
						name = "Admin"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.unleash_role.admin_role", "id", "1"),
					resource.TestCheckResourceAttr("data.unleash_role.admin_role", "name", "Admin"),
					resource.TestCheckResourceAttr("data.unleash_role.admin_role", "type", "root"),
				),
			},
			{
				Config: `
					data "unleash_role" "editor_role" {
						name = "Editor"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.unleash_role.editor_role", "id", "2"),
					resource.TestCheckResourceAttr("data.unleash_role.editor_role", "name", "Editor"),
					resource.TestCheckResourceAttr("data.unleash_role.editor_role", "type", "root"),
				),
			},
			{
				Config: `
					data "unleash_role" "project_member_role" {
						name = "Member"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.unleash_role.project_member_role", "id", "5"),
					resource.TestCheckResourceAttr("data.unleash_role.project_member_role", "name", "Member"),
					resource.TestCheckResourceAttr("data.unleash_role.project_member_role", "type", "project"),
				),
			},
		},
	})
}
