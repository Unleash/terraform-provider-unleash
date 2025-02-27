package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnvironmentResource(t *testing.T) {
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
		},
	})
}
