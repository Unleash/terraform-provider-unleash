package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentResource(t *testing.T) {
	skipUnlessEnterpriseCompatiblePlan(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				//basic creation
				Config: `
					resource "unleash_environment" "fynbos_environment" {
						name = "fynbos"
						type = "semi-arid"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_environment.fynbos_environment", "name", "fynbos"),
					resource.TestCheckResourceAttr("unleash_environment.fynbos_environment", "type", "semi-arid"),
				),
			},
			{
				//modify type
				Config: `
					resource "unleash_environment" "fynbos_environment" {
						name = "fynbos"
						type = "shrubland"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_environment.fynbos_environment", "name", "fynbos"),
					resource.TestCheckResourceAttr("unleash_environment.fynbos_environment", "type", "shrubland"),
				),
			},
			{
				//modify name - makes a new environment
				Config: `
					resource "unleash_environment" "fynbos_environment" {
						name = "nama_karoo"
						type = "semi-desert"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_environment.fynbos_environment", "name", "nama_karoo"),
					resource.TestCheckResourceAttr("unleash_environment.fynbos_environment", "type", "semi-desert"),
				),
			},
			{
				ResourceName:                         "unleash_environment.fynbos_environment",
				ImportStateId:                        "nama_karoo",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccEnvironmentResourceWithRequiredApprovals(t *testing.T) {
	skipUnlessEnterpriseCompatiblePlan(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				//created straight away with environment level change requests
				Config: `
					resource "unleash_environment" "karoo_environment" {
						name               = "succulent_karoo"
						type               = "semi-desert"
						required_approvals = 3
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_environment.karoo_environment", "name", "succulent_karoo"),
					resource.TestCheckResourceAttr("unleash_environment.karoo_environment", "required_approvals", "3"),
				),
			},
			{
				ResourceName:                         "unleash_environment.karoo_environment",
				ImportStateId:                        "succulent_karoo",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
			{
				//modify the number of required approvals
				Config: `
					resource "unleash_environment" "karoo_environment" {
						name               = "succulent_karoo"
						type               = "semi-desert"
						required_approvals = 5
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_environment.karoo_environment", "required_approvals", "5"),
				),
			},
			{
				//removing the attribute clears the environment level configuration
				Config: `
					resource "unleash_environment" "karoo_environment" {
						name = "succulent_karoo"
						type = "semi-desert"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("unleash_environment.karoo_environment", "required_approvals"),
				),
			},
			{
				//required_approvals must be a valid number of approvals.
				//PlanOnly keeps this from touching state, and the step below restores a
				//valid config so the post-test destroy still plans successfully.
				Config: `
					resource "unleash_environment" "karoo_environment" {
						name               = "succulent_karoo"
						type               = "semi-desert"
						required_approvals = 0
					}
				`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("The required_approvals attribute must be between 1 and 10"),
			},
			{
				//back to a valid configuration
				Config: `
					resource "unleash_environment" "karoo_environment" {
						name               = "succulent_karoo"
						type               = "semi-desert"
						required_approvals = 3
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_environment.karoo_environment", "required_approvals", "3"),
				),
			},
		},
	})
}
