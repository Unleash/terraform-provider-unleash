package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectChangeRequestResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "unleash_project" "galaxy-wide-energy" {
						id = "dysonsphere"
						name = "dysonsphere"
					}

					resource "unleash_environment" "space" {
						name = "outerspace"
						type = "vacuum"
					}

					resource "unleash_project_environment" "approvals" {
						project_id = unleash_project.galaxy-wide-energy.id
						environment_name = unleash_environment.space.name
						change_requests_enabled = true
						required_approvals = 2
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_project_environment.approvals", "change_requests_enabled", "true"),
					resource.TestCheckResourceAttr("unleash_project_environment.approvals", "required_approvals", "2"),
				),
			},
			{
				Config: `
					resource "unleash_project" "galaxy-wide-energy" {
						id = "dysonsphere"
						name = "dysonsphere"
					}

					resource "unleash_environment" "space" {
						name = "outerspace"
						type = "vacuum"
					}

					resource "unleash_project_environment" "approvals" {
						project_id = unleash_project.galaxy-wide-energy.id
						environment_name = unleash_environment.space.name
						change_requests_enabled = true
						required_approvals = 20
					}
				`,
				ExpectError: regexp.MustCompile(`(?s)Error: Invalid required_approvals value.*The required_approvals attribute must be between 1 and 10, but got: 20`),
			},
			{
				Config: `
					resource "unleash_project" "galaxy-wide-energy" {
						id = "dysonsphere"
						name = "dysonsphere"
					}

					resource "unleash_environment" "space" {
						name = "outerspace"
						type = "vacuum"
					}

					resource "unleash_project_environment" "approvals" {
						project_id = unleash_project.galaxy-wide-energy.id
						environment_name = unleash_environment.space.name
						change_requests_enabled = true
						required_approvals = 0
					}
				`,
				ExpectError: regexp.MustCompile(`(?s)Error: Invalid required_approvals value.*The required_approvals attribute must be between 1 and 10, but got: 0`),
			},
			{
				Config: `
					resource "unleash_project" "galaxy-wide-energy" {
						id = "dysonsphere"
						name = "dysonsphere"
					}

					resource "unleash_environment" "space" {
						name = "outerspace"
						type = "vacuum"
					}

					resource "unleash_project_environment" "approvals" {
						project_id = unleash_project.galaxy-wide-energy.id
						environment_name = unleash_environment.space.name
						change_requests_enabled = false
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_project_environment.approvals", "change_requests_enabled", "false"),
					resource.TestCheckNoResourceAttr("unleash_project_environment.approvals", "required_approvals"),
				),
			},
			{
				Config: `
					resource "unleash_project" "galaxy-wide-energy" {
						id = "dysonsphere"
						name = "dysonsphere"
					}

					resource "unleash_environment" "testing" {
						name = "lab-environment"
						type = "testing-environment"
					}

					resource "unleash_project_environment" "approvals" {
						project_id = unleash_project.galaxy-wide-energy.id
						environment_name = unleash_environment.testing.name
						change_requests_enabled = true
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_project_environment.approvals", "change_requests_enabled", "true"),
					// Fresh creation should clamp this value to 1
					resource.TestCheckResourceAttr("unleash_project_environment.approvals", "required_approvals", "1"),
				),
			},
		},
	})
}
