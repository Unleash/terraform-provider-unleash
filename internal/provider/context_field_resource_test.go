package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccContextFieldResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				//basic creation
				Config: `
					resource "unleash_context_field" "cheese_context_field" {
						name = "cheese"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "name", "cheese"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "stickiness", "false"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.#", "0"),
					resource.TestCheckNoResourceAttr("unleash_context_field.cheese_context_field", "description"),
				),
			},
			{
				//update all fields
				Config: `
					resource "unleash_context_field" "cheese_context_field" {
						name = "cheese"
						stickiness = true
						description = "Type of cheese to constrain on"
						legal_values = [
							{
								value = "brie"
								description = "Gooey, delicious"
							}
						]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "name", "cheese"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "stickiness", "true"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "description", "Type of cheese to constrain on"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.#", "1"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.0.value", "brie"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.0.description", "Gooey, delicious"),
				),
			},
			{
				//modify only legal values, remove all others
				Config: `
					resource "unleash_context_field" "cheese_context_field" {
						name = "cheese"
						legal_values = [
							{
								value = "camembert"
								description = "More gooey, more delicious"
							}
						]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "name", "cheese"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "stickiness", "false"),
					resource.TestCheckNoResourceAttr("unleash_context_field.cheese_context_field", "description"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.#", "1"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.0.value", "camembert"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.0.description", "More gooey, more delicious"),
				),
			},
			{
				//update the name of the context field
				Config: `
					resource "unleash_context_field" "cheese_context_field" {
						name = "cheese_monger_stock"
						legal_values = [
							{
								value = "camembert"
								description = "More gooey, more delicious"
							}
						]
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "name", "cheese_monger_stock"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "stickiness", "false"),
					resource.TestCheckNoResourceAttr("unleash_context_field.cheese_context_field", "description"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.#", "1"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.0.value", "camembert"),
					resource.TestCheckResourceAttr("unleash_context_field.cheese_context_field", "legal_values.0.description", "More gooey, more delicious"),
				),
			},
		},
	})
}
