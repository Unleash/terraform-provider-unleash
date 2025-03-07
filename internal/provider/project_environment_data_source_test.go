package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectEnvironmentDataSource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "unleash_project_environment" "default_dev" {
						project_id = "default"
						environment_name = "development"
					}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.unleash_project_environment.default_dev", "project_id", "default"),
					resource.TestCheckResourceAttr("data.unleash_project_environment.default_dev", "change_requests_enabled", "false"),
					resource.TestCheckNoResourceAttr("data.unleash_project_environment.default_dev", "required_approvals"),
				),
			},
		},
	})
}
