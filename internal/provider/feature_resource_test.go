package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
)

func TestParseFeatureImportId(t *testing.T) {
	project, name, err := parseFeatureImportId("my.flag")
	assert.NoError(t, err)
	assert.Equal(t, "default", project)
	assert.Equal(t, "my.flag", name)

	project, name, err = parseFeatureImportId("payments/checkout-v2")
	assert.NoError(t, err)
	assert.Equal(t, "payments", project)
	assert.Equal(t, "checkout-v2", name)

	for _, invalid := range []string{"", "/flag", "project/", "a/b/c"} {
		_, _, err = parseFeatureImportId(invalid)
		assert.Error(t, err, invalid)
	}
}

// testAccFeatureAPI talks to the same Unleash instance as the provider, to change or inspect flags
// behind Terraform's back.
func testAccFeatureAPI(t *testing.T) featureAPI {
	t.Helper()
	testAccPreCheck(t)
	var config UnleashConfiguration
	var diags diag.Diagnostics
	client := unleashClient(context.Background(), &UnleashProvider{version: "test"}, &config, &diags)
	if diags.HasError() {
		t.Fatalf("could not configure the Unleash client: %v", diags)
	}
	return featureAPI{client: client}
}

func testAccCheckFeatureArchived(t *testing.T, name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		_, res, _ := testAccFeatureAPI(t).get(context.Background(), "default", name)
		if res == nil || res.StatusCode != http.StatusNotFound {
			return fmt.Errorf("expected feature %s to be archived", name)
		}
		return nil
	}
}

func TestAccFeatureResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// minimal flag, defaults filled in
				Config: `
					resource "unleash_feature" "kill_switch" {
						name = "tf.feature.kill-switch"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_feature.kill_switch", "id", "tf.feature.kill-switch"),
					resource.TestCheckResourceAttr("unleash_feature.kill_switch", "project", "default"),
					resource.TestCheckResourceAttr("unleash_feature.kill_switch", "type", "release"),
					resource.TestCheckResourceAttr("unleash_feature.kill_switch", "impression_data", "false"),
					resource.TestCheckResourceAttr("unleash_feature.kill_switch", "archive_on_destroy", "true"),
					resource.TestCheckNoResourceAttr("unleash_feature.kill_switch", "description"),
				),
			},
			{
				// metadata is updated in place
				Config: `
					resource "unleash_feature" "kill_switch" {
						name            = "tf.feature.kill-switch"
						type            = "kill-switch"
						description     = "Turns off checkout when payments are degraded"
						impression_data = true
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("unleash_feature.kill_switch", "type", "kill-switch"),
					resource.TestCheckResourceAttr("unleash_feature.kill_switch", "description", "Turns off checkout when payments are degraded"),
					resource.TestCheckResourceAttr("unleash_feature.kill_switch", "impression_data", "true"),
				),
			},
			{
				ResourceName:      "unleash_feature.kill_switch",
				ImportState:       true,
				ImportStateId:     "default/tf.feature.kill-switch",
				ImportStateVerify: true,
			},
			{
				// removing the description clears it
				Config: `
					resource "unleash_feature" "kill_switch" {
						name = "tf.feature.kill-switch"
						type = "kill-switch"
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("unleash_feature.kill_switch", "description"),
					resource.TestCheckResourceAttr("unleash_feature.kill_switch", "impression_data", "false"),
				),
			},
			{
				// a flag archived outside Terraform is recreated
				PreConfig: func() {
					_, err := testAccFeatureAPI(t).archive(context.Background(), "default", "tf.feature.kill-switch")
					assert.NoError(t, err)
					_, err = testAccFeatureAPI(t).deleteArchived(context.Background(), "tf.feature.kill-switch")
					assert.NoError(t, err)
				},
				Config: `
					resource "unleash_feature" "kill_switch" {
						name = "tf.feature.kill-switch"
						type = "kill-switch"
					}
				`,
				Check: resource.TestCheckResourceAttr("unleash_feature.kill_switch", "type", "kill-switch"),
			},
			{
				// destroying with archive_on_destroy = false also frees the name
				Config: `
					resource "unleash_feature" "kill_switch" {
						name               = "tf.feature.kill-switch"
						type               = "kill-switch"
						archive_on_destroy = false
					}
				`,
			},
		},
		CheckDestroy: func(*terraform.State) error {
			// the name can be reused because the flag was deleted from the archive too
			api := testAccFeatureAPI(t)
			if _, err := api.create(context.Background(), "default", createFeatureRequest{Name: "tf.feature.kill-switch"}); err != nil {
				return fmt.Errorf("expected the flag name to be free after destroy: %w", err)
			}
			_, _ = api.archive(context.Background(), "default", "tf.feature.kill-switch")
			_, _ = api.deleteArchived(context.Background(), "tf.feature.kill-switch")
			return nil
		},
	})
}

func TestAccFeatureResourceArchivesOnDestroy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "unleash_feature" "archived" {
						name = "tf.feature.archived"
					}
				`,
			},
		},
		CheckDestroy: func(s *terraform.State) error {
			if err := testAccCheckFeatureArchived(t, "tf.feature.archived")(s); err != nil {
				return err
			}
			// it went to the archive rather than being deleted, so the name is still taken until it's purged
			api := testAccFeatureAPI(t)
			_, err := api.create(context.Background(), "default", createFeatureRequest{Name: "tf.feature.archived"})
			_, _ = api.deleteArchived(context.Background(), "tf.feature.archived")
			if err == nil {
				return fmt.Errorf("expected the archived flag to still hold its name")
			}
			return nil
		},
	})
}

func TestAccFeatureResourceRejectsInvalidName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "unleash_feature" "invalid" {
						name = "not a valid name"
					}
				`,
				ExpectError: regexp.MustCompile(`must only contain letters, numbers and the characters -._~`),
			},
		},
	})
}

func TestAccFeatureDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					resource "unleash_feature" "source" {
						name               = "tf.feature.data-source"
						type               = "operational"
						description        = "Read back through the data source"
						archive_on_destroy = false
					}

					data "unleash_feature" "read" {
						name = unleash_feature.source.name
					}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.unleash_feature.read", "project", "default"),
					resource.TestCheckResourceAttr("data.unleash_feature.read", "type", "operational"),
					resource.TestCheckResourceAttr("data.unleash_feature.read", "description", "Read back through the data source"),
					resource.TestCheckResourceAttr("data.unleash_feature.read", "impression_data", "false"),
					resource.TestCheckResourceAttr("data.unleash_feature.read", "stale", "false"),
					// a new flag exists, disabled, in every environment of the project
					resource.TestCheckResourceAttr("data.unleash_feature.read", "environments.development", "false"),
					resource.TestCheckResourceAttr("data.unleash_feature.read", "environments.production", "false"),
				),
			},
		},
	})
}
