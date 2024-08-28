package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccServiceAccountTokenResource(t *testing.T) {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "unleash_service_account" "account_for_tokens_test" {
    				name = "something"
					username = "also_something_probably"
					root_role = 1
				}

				resource "unleash_service_account_token" "token_for_account_test" {
					service_account_id = unleash_service_account.account_for_tokens_test.id
					description = "a token for the account"
					expires_at = "2048-01-01T00:00:00Z"
				}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_service_account.account_for_tokens_test", "id"),
					resource.TestCheckResourceAttrSet("unleash_service_account_token.token_for_account_test", "id"),
					resource.TestCheckResourceAttrSet("unleash_service_account_token.token_for_account_test", "secret"),
					resource.TestCheckResourceAttr("unleash_service_account_token.token_for_account_test", "description", "a token for the account"),
					resource.TestCheckResourceAttr("unleash_service_account_token.token_for_account_test", "expires_at", "2048-01-01T00:00:00Z"),
				),
			},
		},
	})
}

func TestAccServiceAccountTokenResourceUpdatingDescriptionGeneratesNewToken(t *testing.T) {

	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		t.Skip("Skipping enterprise tests")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
						resource "unleash_service_account" "account_for_tokens_test" {
							name = "something"
							username = "also_something_probably"
							root_role = 1
						}

						resource "unleash_service_account_token" "token_for_account_test" {
							service_account_id = unleash_service_account.account_for_tokens_test.id
							description = "a token for the account"
							expires_at = "2048-01-01T00:00:00Z"
						}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_service_account.account_for_tokens_test", "id"),
					resource.TestCheckResourceAttrSet("unleash_service_account_token.token_for_account_test", "id"),
					resource.TestCheckResourceAttrSet("unleash_service_account_token.token_for_account_test", "secret"),
					resource.TestCheckResourceAttr("unleash_service_account_token.token_for_account_test", "description", "a token for the account"),
					resource.TestCheckResourceAttr("unleash_service_account_token.token_for_account_test", "expires_at", "2048-01-01T00:00:00Z"),
				),
			},
			{
				// Update the description here to trigger an update. We don't support updates because out API doesn't support
				// updates BUT we've marked this property as triggering a plan change, so this should force terraform
				// to delete and regenerate the new token without encountering the error state in the update method
				Config: `
						resource "unleash_service_account" "account_for_tokens_test" {
							name = "something"
							username = "also_something_probably"
							root_role = 1
						}

						resource "unleash_service_account_token" "token_for_account_test" {
							service_account_id = unleash_service_account.account_for_tokens_test.id
							description = "updated token description"
							expires_at = "2048-01-01T00:00:00Z"
						}
					`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_service_account_token.token_for_account_test", "id"),
					resource.TestCheckResourceAttrSet("unleash_service_account_token.token_for_account_test", "secret"),
					resource.TestCheckResourceAttr("unleash_service_account_token.token_for_account_test", "description", "updated token description"),
					resource.TestCheckResourceAttr("unleash_service_account_token.token_for_account_test", "expires_at", "2048-01-01T00:00:00Z"),
					testCheckDifferentSecrets("unleash_service_account_token.token_for_account_test"),
				),
			},
		},
	})
}

// Little bit of terraform testing magic. From the actual test it looks like this is only called once,
// but it isn't, it's called on each step, so it gives us a chance to trap the secret on the first
// step and compare on the second
func testCheckDifferentSecrets(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		secret, ok := resource.Primary.Attributes["secret"]
		if !ok {
			return fmt.Errorf("secret not found in state for resource: %s", resourceName)
		}

		if lastSecret == "" {
			lastSecret = secret
			return nil
		}

		if secret == lastSecret {
			return fmt.Errorf("expected a new secret to be generated, but the secret did not change")
		}

		return nil
	}
}

var lastSecret string
