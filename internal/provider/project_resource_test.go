package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
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

func testAccSampleProjectResourceWithMode(name string, id string, mode string) string {
	return fmt.Sprintf(`
		resource "unleash_project" "test_project3" {
		id = "%s"
		name = "%s"
		description = "test description"
		mode = "%s"
	}`, id, name, mode)
}

func testAccSampleProjectResourceWithEnterpriseSettings(name string, id string) string {
	return fmt.Sprintf(`
		resource "unleash_project" "project_enterprise" {
			id = "%s"
			name = "%s"
			description = "project with enterprise settings"

			feature_naming = {
				pattern     = "^[a-z0-9_-]+$"
				example     = "example-flag"
				description = "Only lowercase letters, numbers, underscores, and dashes."
			}

			link_templates = [
				{
					title        = "Docs"
					url_template = "https://example.com/projects/{{project}}/features/{{feature}}"
				},
				{
					title        = "Tracker"
					url_template = "https://example.com/tracker/{{feature}}"
				}
			]
		}`, id, name)
}

func testAccSampleProjectResourceWithUpdatedEnterpriseSettings(name string, id string) string {
	return fmt.Sprintf(`
		resource "unleash_project" "project_enterprise" {
			id = "%s"
			name = "%s"
			description = "project with updated enterprise settings"

			feature_naming = {
				pattern     = "^feature_[a-z0-9]+$"
				example     = "feature_name"
				description = "Feature names must start with feature_."
			}

			link_templates = [
				{
					title        = "Docs"
					url_template = "https://example.com/docs/{{project}}/{{feature}}"
				}
			]
		}`, id, name)
}

func TestAccProjectResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}

	randomID := func() string {
		return fmt.Sprintf("tf-acc-%s", strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)))
	}
	randomName := func(prefix string) string {
		return fmt.Sprintf("%s-%s", prefix, strings.ToLower(acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)))
	}

	projectID := randomID()
	projectName := randomName("TestProject")
	projectRenamed := randomName("Renamed")

	projectNoDescID := randomID()
	projectNoDescName := randomName("NoDescription")

	projectModeID := randomID()
	projectModeName := randomName("ModeProject")

	projectEnterpriseID := randomID()
	projectEnterpriseName := randomName("EnterpriseProject")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSampleProjectResource(projectName, projectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project", "name"),
					resource.TestCheckResourceAttr("unleash_project.test_project", "name", projectName),
					resource.TestCheckResourceAttr("unleash_project.test_project", "description", "test description"),
				),
			},
			{
				Config: testAccSampleProjectResource(projectRenamed, projectID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project", "name"),
					resource.TestCheckResourceAttr("unleash_project.test_project", "name", projectRenamed),
					resource.TestCheckResourceAttr("unleash_project.test_project", "description", "test description"),
				),
			},
			{
				Config: testAccSampleProjectResourceWithNoDescription(projectNoDescName, projectNoDescID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project2", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project2", "name"),
				),
			},
			{
				Config:            `resource "unleash_project" "newly_imported" {}`,
				ImportStateId:     projectNoDescID,
				ResourceName:      "unleash_project.newly_imported",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccSampleProjectResourceWithMode(projectModeName, projectModeID, "open"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project3", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project3", "name"),
					resource.TestCheckResourceAttr("unleash_project.test_project3", "name", projectModeName),
					resource.TestCheckResourceAttr("unleash_project.test_project3", "mode", "open"),
				),
			},
			{
				Config: testAccSampleProjectResourceWithMode(projectModeName, projectModeID, "private"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_project.test_project3", "id"),
					resource.TestCheckResourceAttrSet("unleash_project.test_project3", "name"),
					resource.TestCheckResourceAttr("unleash_project.test_project3", "name", projectModeName),
					resource.TestCheckResourceAttr("unleash_project.test_project3", "mode", "private"),
				),
			},
			{
				Config: testAccSampleProjectResourceWithEnterpriseSettings(projectEnterpriseName, projectEnterpriseID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "feature_naming.pattern", "^[a-z0-9_-]+$"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "feature_naming.example", "example-flag"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "feature_naming.description", "Only lowercase letters, numbers, underscores, and dashes."),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "name", projectEnterpriseName),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "link_templates.#", "2"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "link_templates.0.title", "Docs"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "link_templates.0.url_template", "https://example.com/projects/{{project}}/features/{{feature}}"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "link_templates.1.title", "Tracker"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "link_templates.1.url_template", "https://example.com/tracker/{{feature}}"),
				),
			},
			{
				Config: testAccSampleProjectResourceWithUpdatedEnterpriseSettings(projectEnterpriseName, projectEnterpriseID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "feature_naming.pattern", "^feature_[a-z0-9]+$"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "feature_naming.example", "feature_name"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "feature_naming.description", "Feature names must start with feature_."),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "name", projectEnterpriseName),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "link_templates.#", "1"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "link_templates.0.title", "Docs"),
					resource.TestCheckResourceAttr("unleash_project.project_enterprise", "link_templates.0.url_template", "https://example.com/docs/{{project}}/{{feature}}"),
				),
			},
		},
	})
}
