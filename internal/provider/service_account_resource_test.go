package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func makeServiceAccountDef(name string, user_name string, root_role int32) string {
	return fmt.Sprintf(`
		resource "unleash_service_account" "test_service_account" {
    	name = "%s"
		username = "%s"
		root_role = %d
	}`, name, user_name, root_role)
}

func TestAccServiceAccountResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: makeServiceAccountDef("test_service_account", "test_user", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_service_account.test_service_account", "id"),
					resource.TestCheckResourceAttr("unleash_service_account.test_service_account", "name", "test_service_account"),
					resource.TestCheckResourceAttr("unleash_service_account.test_service_account", "username", "test_user"),
					resource.TestCheckResourceAttr("unleash_service_account.test_service_account", "root_role", "1"),
				),
			},
		},
	})
}
