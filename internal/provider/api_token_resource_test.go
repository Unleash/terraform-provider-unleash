package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestApiTokenNameRequiresReplacement(t *testing.T) {
	ctx := context.Background()
	var schemaResp frameworkresource.SchemaResponse
	NewApiTokenResource().Schema(ctx, frameworkresource.SchemaRequest{}, &schemaResp)
	attribute, ok := schemaResp.Schema.Attributes["token_name"].(schema.StringAttribute)
	if !ok {
		t.Fatal("token_name must be a string attribute")
	}
	if len(attribute.PlanModifiers) == 0 {
		t.Fatal("token_name must have a replacement plan modifier")
	}

	for _, tc := range []struct {
		name            string
		create          bool
		destroy         bool
		unknownName     bool
		state           string
		plan            string
		requiresReplace bool
	}{
		{name: "rename", state: "old_token", plan: "new_token", requiresReplace: true},
		{name: "unchanged", state: "old_token", plan: "old_token"},
		{name: "creation", plan: "new_token", create: true},
		{name: "destruction", state: "old_token", destroy: true},
		{name: "unknown name", state: "old_token", unknownName: true, requiresReplace: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rawType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"token_name": tftypes.String}}
			req := planmodifier.StringRequest{
				Path: path.Root("token_name"),
				State: tfsdk.State{Raw: tftypes.NewValue(rawType, map[string]tftypes.Value{
					"token_name": tftypes.NewValue(tftypes.String, tc.state),
				})},
				Plan: tfsdk.Plan{Raw: tftypes.NewValue(rawType, map[string]tftypes.Value{
					"token_name": tftypes.NewValue(tftypes.String, tc.plan),
				})},
				StateValue: types.StringValue(tc.state),
				PlanValue:  types.StringValue(tc.plan),
			}
			resp := planmodifier.StringResponse{PlanValue: req.PlanValue}
			if tc.create {
				req.State.Raw = tftypes.NewValue(rawType, nil)
			}
			if tc.destroy {
				req.Plan.Raw = tftypes.NewValue(rawType, nil)
			}
			if tc.unknownName {
				req.PlanValue = types.StringUnknown()
			}
			for _, modifier := range attribute.PlanModifiers {
				modifier.PlanModifyString(ctx, req, &resp)
			}
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected plan diagnostics: %v", resp.Diagnostics)
			}
			if resp.RequiresReplace != tc.requiresReplace {
				t.Fatalf("RequiresReplace = %t, want %t", resp.RequiresReplace, tc.requiresReplace)
			}
			if tc.requiresReplace && resp.Diagnostics.WarningsCount() != 1 {
				t.Fatal("replacement must warn that the old token will be revoked")
			}
			if !tc.requiresReplace && resp.Diagnostics.WarningsCount() != 0 {
				t.Fatal("creation, destruction, and unchanged names must not produce rename warnings")
			}
		})
	}
}

func TestAccApiTokenResource(t *testing.T) {
	secure := os.Getenv("SECURE_TOKEN") == "true"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
				resource "unleash_api_token" "frontend_token" {
					token_name = "frontend_token"
					type = "frontend"
					expires_at = "2024-12-31T23:59:59Z"
					projects = ["*"]
					environment = "development"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.frontend_token", "secret"),
					resource.TestCheckResourceAttrSet("unleash_api_token.frontend_token", "expires_at"),
					resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "token_name", "frontend_token"),
					resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "environment", "development"),
					resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "projects.0", "*"),
					resource.TestCheckResourceAttr("unleash_api_token.frontend_token", "secure", func() string {
						if secure {
							return "true"
						}

						return "false"
					}()),
				),
			},
			{
				Config: `
				resource "unleash_api_token" "client_token" {
					token_name = "client_token"
					type = "client"
					expires_at = "2024-12-31T23:59:59Z"
					projects = ["default"]
					environment = "development"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.client_token", "secret"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "expires_at", "2024-12-31T23:59:59Z"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "token_name", "client_token"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "environment", "development"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "projects.0", "default"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "secure", func() string {
						if secure {
							return "true"
						}

						return "false"
					}()),
				),
			},
			{ // test change expire date for previous token
				Config: `
				resource "unleash_api_token" "client_token" {
					token_name = "client_token"
					type = "client"
					expires_at = "2025-01-01T12:00:00Z"
					projects = ["default"]
					environment = "development"
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.client_token", "secret"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "expires_at", "2025-01-01T12:00:00Z"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "token_name", "client_token"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "environment", "development"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "projects.0", "default"),
					resource.TestCheckResourceAttr("unleash_api_token.client_token", "secure", func() string {
						if secure {
							return "true"
						}

						return "false"
					}()),
				),
			},
			{
				Config: `
				resource "unleash_api_token" "client_no_expire" {
					token_name = "client_no_expire"
					type = "client"
					projects = ["default"]
				}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("unleash_api_token.client_no_expire", "secret"),
					resource.TestCheckNoResourceAttr("unleash_api_token.client_no_expire", "expires_at"),
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "token_name", "client_no_expire"),
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "environment", func() string {
						if v := os.Getenv("DEFAULT_ENVIRONMENT"); v != "" {
							return v
						}

						return "development"
					}()),
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "projects.0", "default"),
					resource.TestCheckResourceAttr("unleash_api_token.client_no_expire", "secure", func() string {
						if secure {
							return "true"
						}

						return "false"
					}()),
				),
			},
		},
	})
}
