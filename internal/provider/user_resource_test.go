package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

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
	// TODO retrieve the client from Provider configuration rather than creating a new client
	configuration := unleash.NewConfiguration()
	base_url := os.Getenv("UNLEASH_URL")
	authorization := os.Getenv("AUTH_TOKEN")
	configuration.Servers = unleash.ServerConfigurations{
		{
			URL: base_url,
		},
	}
	configuration.HTTPClient = httpClient(false)
	configuration.AddDefaultHeader("Authorization", authorization)
	apiClient := unleash.NewAPIClient(configuration)

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
