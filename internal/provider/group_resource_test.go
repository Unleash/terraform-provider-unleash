package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func customCheckGroupUserExists(resourceName string, userID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		for i := 0; ; i++ {
			userAttrKey := fmt.Sprintf("users.%d", i)
			user, ok := rs.Primary.Attributes[userAttrKey]
			if !ok {
				break // Exit the loop if no more users are found
			}
			if user == userID {
				return nil
			}
		}

		return fmt.Errorf("User ID %s not found in resource %s", userID, resourceName)
	}
}

func customCheckGroupMappingSSOExists(resourceName string, mapping string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		for i := 0; ; i++ {
			mappingAttrKey := fmt.Sprintf("mappings_sso.%d", i)
			m, ok := rs.Primary.Attributes[mappingAttrKey]
			if !ok {
				break // Exit the loop if no more mappings are found
			}
			if m == mapping {
				return nil
			}
		}

		return fmt.Errorf("SSO mapping %s not found in resource %s", mapping, resourceName)
	}
}

func TestAccGroupResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test 1: Create basic group with just name
			{
				Config: `
				resource "unleash_group" "test_group" {
					name = "Test Group"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.test_group", "id"),
					resource.TestCheckResourceAttr("unleash_group.test_group", "name", "Test Group"),
				),
			},
			// Test 2: Update group to add description
			{
				Config: `
				resource "unleash_group" "test_group" {
					name        = "Test Group"
					description = "A test group for Terraform provider"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.test_group", "id"),
					resource.TestCheckResourceAttr("unleash_group.test_group", "name", "Test Group"),
					resource.TestCheckResourceAttr("unleash_group.test_group", "description", "A test group for Terraform provider"),
				),
			},
			// Test 3: Update group name
			{
				Config: `
				resource "unleash_group" "test_group" {
					name        = "Renamed Test Group"
					description = "A test group for Terraform provider"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.test_group", "id"),
					resource.TestCheckResourceAttr("unleash_group.test_group", "name", "Renamed Test Group"),
					resource.TestCheckResourceAttr("unleash_group.test_group", "description", "A test group for Terraform provider"),
				),
			},
			// Test 4: Add SSO mappings
			{
				Config: `
				resource "unleash_group" "test_group" {
					name         = "Renamed Test Group"
					description  = "A test group for Terraform provider"
					mappings_sso = ["SSOGroup1", "SSOGroup2"]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.test_group", "id"),
					resource.TestCheckResourceAttr("unleash_group.test_group", "name", "Renamed Test Group"),
					customCheckGroupMappingSSOExists("unleash_group.test_group", "SSOGroup1"),
					customCheckGroupMappingSSOExists("unleash_group.test_group", "SSOGroup2"),
				),
			},
			// Test 5: Update SSO mappings
			{
				Config: `
				resource "unleash_group" "test_group" {
					name         = "Renamed Test Group"
					description  = "A test group for Terraform provider"
					mappings_sso = ["SSOGroup3"]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.test_group", "id"),
					resource.TestCheckResourceAttr("unleash_group.test_group", "name", "Renamed Test Group"),
					customCheckGroupMappingSSOExists("unleash_group.test_group", "SSOGroup3"),
				),
			},
			// Test 6: Create group with users (requires valid user IDs in your Unleash instance)
			{
				Config: `
				resource "unleash_group" "group_with_users" {
					name        = "Group with Users"
					description = "Testing user assignments"
					users       = [123]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.group_with_users", "id"),
					resource.TestCheckResourceAttr("unleash_group.group_with_users", "name", "Group with Users"),
					customCheckGroupUserExists("unleash_group.group_with_users", "123"),
				),
			},
			// Test 7: Update users list
			{
				Config: `
				resource "unleash_group" "group_with_users" {
					name        = "Group with Users"
					description = "Testing user assignments"
					users       = [123,456]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.group_with_users", "id"),
					resource.TestCheckResourceAttr("unleash_group.group_with_users", "name", "Group with Users"),
					customCheckGroupUserExists("unleash_group.group_with_users", "123"),
					customCheckGroupUserExists("unleash_group.group_with_users", "456"),
				),
			},
			// Test 8: Remove users
			{
				Config: `
				resource "unleash_group" "group_with_users" {
					name        = "Group with Users"
					description = "Testing user assignments"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.group_with_users", "id"),
					resource.TestCheckResourceAttr("unleash_group.group_with_users", "name", "Group with Users"),
					resource.TestCheckResourceAttr("unleash_group.group_with_users", "users.#", "0"),
				),
			},
			// Test 9: Create group with root role
			{
				Config: `
				resource "unleash_group" "group_with_role" {
					name        = "Group with Root Role"
					description = "Testing root role assignment"
					root_role   = 1
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.group_with_role", "id"),
					resource.TestCheckResourceAttr("unleash_group.group_with_role", "name", "Group with Root Role"),
					resource.TestCheckResourceAttr("unleash_group.group_with_role", "root_role", "1"),
				),
			},
			// Test 10: Update root role
			{
				Config: `
				resource "unleash_group" "group_with_role" {
					name        = "Group with Root Role"
					description = "Testing root role assignment"
					root_role   = 2
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.group_with_role", "id"),
					resource.TestCheckResourceAttr("unleash_group.group_with_role", "name", "Group with Root Role"),
					resource.TestCheckResourceAttr("unleash_group.group_with_role", "root_role", "2"),
				),
			},
			// Test 11: Create comprehensive group with all fields
			{
				Config: `
				resource "unleash_group" "comprehensive_group" {
					name         = "Comprehensive Group"
					description  = "Group with all fields populated"
					root_role    = 1
					mappings_sso = ["AdminGroup", "DevGroup"]
					users        = [123]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.comprehensive_group", "id"),
					resource.TestCheckResourceAttr("unleash_group.comprehensive_group", "name", "Comprehensive Group"),
					resource.TestCheckResourceAttr("unleash_group.comprehensive_group", "description", "Group with all fields populated"),
					resource.TestCheckResourceAttr("unleash_group.comprehensive_group", "root_role", "1"),
					customCheckGroupMappingSSOExists("unleash_group.comprehensive_group", "AdminGroup"),
					customCheckGroupMappingSSOExists("unleash_group.comprehensive_group", "DevGroup"),
					customCheckGroupUserExists("unleash_group.comprehensive_group", "123"),
				),
			},
			// Test 12: Import state
			{
				ResourceName:      "unleash_group.comprehensive_group",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccGroupResource_MinimalConfig tests creating a group with minimal configuration.
func TestAccGroupResource_MinimalConfig(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "unleash_group" "minimal" {
					name = "Minimal Group"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.minimal", "id"),
					resource.TestCheckResourceAttr("unleash_group.minimal", "name", "Minimal Group"),
				),
			},
		},
	})
}

// TestAccGroupResource_RemoveOptionalFields tests removing optional fields.
func TestAccGroupResource_RemoveOptionalFields(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "unleash_group" "optional_fields" {
					name         = "Group with Optionals"
					description  = "Has description"
					mappings_sso = ["Mapping1"]
					root_role    = 1
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.optional_fields", "id"),
					resource.TestCheckResourceAttr("unleash_group.optional_fields", "description", "Has description"),
					resource.TestCheckResourceAttr("unleash_group.optional_fields", "root_role", "1"),
				),
			},
			{
				Config: `
				resource "unleash_group" "optional_fields" {
					name = "Group with Optionals"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_group.optional_fields", "id"),
					resource.TestCheckResourceAttr("unleash_group.optional_fields", "name", "Group with Optionals"),
					// Check that optional fields are removed/null
					resource.TestCheckNoResourceAttr("unleash_group.optional_fields", "description"),
					resource.TestCheckResourceAttr("unleash_group.optional_fields", "mappings_sso.#", "0"),
				),
			},
		},
	})
}
