package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectAccessResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// TODO with set access we can mess up the project access and it can end up without an owner which should not be allowed. Add this restriction in the backend
				// Note: from the UI I can add a member role and then remove the owner leaving a project ownerless but with members.
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
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "project", "sample"),
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "roles.#", "1"),
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "roles.0.groups.#", "0"),
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "roles.0.users.#", "1"),
				),
			},
			{
				// Update previous configuration to add a new member and promoting test to owner
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

					resource "unleash_user" "test_user_2" {
						name = "tester-2"
						email = "test-2-password@getunleash.io"
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
									unleash_user.test_user.id
								]
								groups = []
							},
							{
								role = data.unleash_role.project_member_role.id
								users = [
									unleash_user.test_user_2.id
								]
								groups = []
							},
						]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "project", "sample"),
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "roles.#", "2"),
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "roles.0.groups.#", "0"),
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "roles.1.groups.#", "0"),
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "roles.0.users.#", "1"),
					resource.TestCheckResourceAttr("unleash_project_access.sample_project_access", "roles.1.users.#", "1"),
				),
			},
		},
	})
}
