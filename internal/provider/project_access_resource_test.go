package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectAccessResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") == "false" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "unleash_project" "sample_project" {
						id = "sample"
						name = "sample-project"
					}

					data "unleash_role" "project_owner_role" {
						name = "Owner"
					}

					data "unleash_role" "project_member_role" {
						name = "Member"
					}

					resource "unleash_user" "test_user" {
						name = "tester"
						email = "test-password@getunleash.io"
						password = "you-will-never-guess"
						root_role = "3"
						send_email = false
					}

					resource "unleash_project_access" "sample_project_access" {
						project = unleash_project.sample_project.id
						roles = [
							{
								role = data.unleash_role.project_owner_role.id
								users = [
									1
								]
								groups = []
							},
							{
								role = data.unleash_role.project_member_role.id
								users = [
									unleash_user.test_user.id
								]
								groups = []
							},
						]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project_access.sample_project_access", "project"),
					// resource.TestCheckResourceAttrSet("unleash_project_access.sample_project_access", "name"),
					// resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "name", "TestProjectName"),
					// resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "description", "test description"),
				),
			},
		},
	})
}
