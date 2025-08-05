package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccDeleteUserFromAPI is a custom TestCheckFunc that deletes the user from the API
// This simulates external deletion to test Terraform's ability to handle resources deleted outside of Terraform.
func testAccDeleteUserFromAPI(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		// Create client using the same configuration as the provider
		config := &UnleashConfiguration{
			BaseUrl:       types.StringNull(), // Will use environment variable
			Authorization: types.StringNull(), // Will use environment variable
		}

		var diagnostics diag.Diagnostics
		provider := &UnleashProvider{version: "test"}
		client := unleashClient(context.Background(), provider, config, &diagnostics)

		if diagnostics.HasError() {
			return fmt.Errorf("Failed to create test client: %v", diagnostics.Errors())
		}

		idInt, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Failed to convert user id to int: %v", err)
		}

		id := int32(idInt)
		_, err = client.UsersAPI.DeleteUser(context.Background(), id).Execute()
		if err != nil {
			return fmt.Errorf("Failed to delete user: %v", err)
		}

		return nil
	}
}

func testAccSampleUserResource(name string) string {
	return fmt.Sprintf(`
resource "unleash_user" "the_newbie" {
    name = "%s"
    email = "test@getunleash.io"
    root_role = "2"
    send_email = false
}`, name)
}
func testAccSampleUserResourceWithPassword(resource string, name string) string {
	return fmt.Sprintf(`
resource "unleash_user" "%s" {
    name = "%s"
    email = "test-password@getunleash.io"
	password = "you-will-never-guess"
    root_role = "3"
    send_email = false
}`, resource, name)
}
func TestAccUserResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSampleUserResource("Test User"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_user.the_newbie", "id"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "name", "Test User"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "email", "test@getunleash.io"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "root_role", "2"),
					// TODO test the remote object matches https://developer.hashicorp.com/terraform/plugin/testing/testing-patterns#basic-test-to-verify-attributes
				),
			},
			{
				Config: testAccSampleUserResource("Renamed user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_user.the_newbie", "id"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "name", "Renamed user"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "email", "test@getunleash.io"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "root_role", "2"),
				// TODO test the remote object matches https://developer.hashicorp.com/terraform/plugin/testing/testing-patterns#basic-test-to-verify-attributes
				),
			},
			{
				ExpectNonEmptyPlan: true,
				Config:             testAccSampleUserResource("Renamed user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDeleteUserFromAPI("unleash_user.the_newbie"),
				),
			},
			{
				Config: testAccSampleUserResource("Renamed user"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_user.the_newbie", "id"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "name", "Renamed user"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "email", "test@getunleash.io"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "root_role", "2"),
				),
			},
			{
				Config: testAccSampleUserResourceWithPassword("with_pass", "User with password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_user.with_pass", "id"),
					resource.TestCheckResourceAttr("unleash_user.with_pass", "name", "User with password"),
					resource.TestCheckResourceAttr("unleash_user.with_pass", "email", "test-password@getunleash.io"),
					resource.TestCheckResourceAttr("unleash_user.with_pass", "root_role", "3"),
					resource.TestCheckResourceAttr("unleash_user.with_pass", "password", "you-will-never-guess"),
				// TODO test the remote object matches https://developer.hashicorp.com/terraform/plugin/testing/testing-patterns#basic-test-to-verify-attributes
				),
			},
		},
		CheckDestroy: testAccCheckUserResourceDestroy,
	})
}

func TestAccUserResourceImport(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSampleUserResource("Test User"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_user.the_newbie", "id"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "name", "Test User"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "email", "test@getunleash.io"),
					resource.TestCheckResourceAttr("unleash_user.the_newbie", "root_role", "2"),
					// TODO test the remote object matches https://developer.hashicorp.com/terraform/plugin/testing/testing-patterns#basic-test-to-verify-attributes
				),
			},
			{
				Config:            `resource "unleash_user" "the_newbie" {}`,
				ResourceName:      "unleash_user.the_newbie",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckUserResourceDestroy(s *terraform.State) error {
	// Use the same client configuration as the provider instead of creating a new one
	config := &UnleashConfiguration{
		BaseUrl:       types.StringNull(), // Will use environment variable
		Authorization: types.StringNull(), // Will use environment variable
	}

	var diagnostics diag.Diagnostics
	provider := &UnleashProvider{version: "test"}
	apiClient := unleashClient(context.Background(), provider, config, &diagnostics)

	if diagnostics.HasError() {
		return fmt.Errorf("Failed to create test client: %v", diagnostics.Errors())
	}

	// loop through the resources in state, verifying each widget
	// is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "unleash_user" {
			continue
		}

		userId, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Expected an integer")
		}

		user, response, err := apiClient.UsersAPI.GetUser(context.Background(), int32(userId)).Execute()
		if err == nil {
			if fmt.Sprintf("%v", user.Id) == rs.Primary.ID {
				return fmt.Errorf("User (%s) still exists.", rs.Primary.ID)
			}

			return nil
		}

		// If the error is equivalent to 404 not found, the widget is destroyed.
		// Otherwise return the error
		if response.StatusCode != 404 {
			return fmt.Errorf("Invalid response code %d. Expected 404", response.StatusCode)
		}
	}

	return nil
}
