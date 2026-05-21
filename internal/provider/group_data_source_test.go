package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupDataSource(t *testing.T) {
	skipUnlessEnterpriseCompatiblePlan(t)
	groupName := "Lookup Group " + acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "unleash_group" "lookup_group" {
					name         = "` + groupName + `"
					description  = "Group used by data source acceptance tests"
					mappings_sso = ["LookupSSOGroup"]
					root_role    = 1
				}

				data "unleash_group" "by_name" {
					name = unleash_group.lookup_group.name
				}

				data "unleash_group" "by_id" {
					id = unleash_group.lookup_group.id
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unleash_group.by_name", "id"),
					resource.TestCheckResourceAttr("data.unleash_group.by_name", "name", groupName),
					resource.TestCheckResourceAttr("data.unleash_group.by_name", "description", "Group used by data source acceptance tests"),
					resource.TestCheckResourceAttr("data.unleash_group.by_name", "mappings_sso.#", "1"),
					resource.TestCheckResourceAttr("data.unleash_group.by_name", "root_role", "1"),
					resource.TestCheckResourceAttrPair("data.unleash_group.by_id", "id", "unleash_group.lookup_group", "id"),
					resource.TestCheckResourceAttr("data.unleash_group.by_id", "name", groupName),
				),
			},
		},
	})
}
