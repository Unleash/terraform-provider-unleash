package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSampleProjectResource(name string, id string) string {
	return fmt.Sprintf(`
		resource "unleash_project" "test_project" {
		id = "%s"
    	name = "%s"
		description = "test description"
	}`, id, name)
}

func testAccSampleProjectResourceWithNoDescription(name string, id string) string {
	return fmt.Sprintf(`
		resource "unleash_project" "test_project2" {
		id = "%s"
    	name = "%s"
	}`, id, name)
}

func testAccSampleProjectResourceWithMode(id string, mode string) string {
	return fmt.Sprintf(`
		resource "unleash_project" "test_project3" {
		id = "%s"
		name = "TestProjectName"
		description = "test description"
		mode = "%s"
	}`, id, mode)
}

func TestAccProjectResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSampleProjectResource("TestProjectName", "TestId"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project", "name"),
					resource.TestCheckResourceAttr("unleash_project.test_project", "name", "TestProjectName"),
					resource.TestCheckResourceAttr("unleash_project.test_project", "description", "test description"),
				),
			},
			{
				Config: testAccSampleProjectResource("RenamedToThisString", "TestId"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project", "name"),
					resource.TestCheckResourceAttr("unleash_project.test_project", "name", "RenamedToThisString"),
					resource.TestCheckResourceAttr("unleash_project.test_project", "description", "test description"),
				),
			},
			{
				Config: testAccSampleProjectResourceWithNoDescription("NoDescription", "TestId2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project2", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project2", "name"),
				),
			},
			{
				Config:            `resource "unleash_project" "newly_imported" {}`,
				ImportStateId:     "TestId2",
				ResourceName:      "unleash_project.newly_imported",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccSampleProjectResourceWithMode("TestId3", "open"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project3", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project3", "name"),
					resource.TestCheckResourceAttr("unleash_project.test_project3", "mode", "open"),
				),
			},
			{
				Config: testAccSampleProjectResourceWithMode("TestId3", "private"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project3", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project3", "name"),
					resource.TestCheckResourceAttr("unleash_project.test_project3", "mode", "private"),
				),
			},
		},
	})
}
