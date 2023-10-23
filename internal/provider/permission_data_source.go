package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &permissionDataSource{}
	_ datasource.DataSourceWithConfigure = &permissionDataSource{}
)

// NewPermissionDataSource is a helper function to simplify the provider implementation.
func NewPermissionDataSource() datasource.DataSource {
	return &permissionDataSource{}
}

// permissionDataSource is the data source implementation.
type permissionDataSource struct {
	client *unleash.APIClient
}

type permissionDataSourceModel struct {
	// The identifier for this permission
	Id types.Int64 `tfsdk:"id"`
	// The name of this permission
	Name types.String `tfsdk:"name"`
	// The name to display in listings of permissions
	DisplayName types.String `tfsdk:"display_name"`
	// What level this permission applies to. Either root, project or the name of the environment it applies to
	Type types.String `tfsdk:"type"`
	// Which environment this permission applies to
	Environment types.String `tfsdk:"environment"`
}

// Configure adds the provider configured client to the data source.
func (d *permissionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	d.client = client

}

// Metadata returns the data source type name.
func (d *permissionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

// Schema defines the schema for the data source. TODO: can we transform OpenAPI schema into TF schema?
func (d *permissionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a permission.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "Identifier for this permission.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the permission.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "What level this permission applies to. Either root, project or the name of the environment it applies to.",
				Computed:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "The name to display in listings of permissions.",
				Computed:    true,
			},
			"environment": schema.StringAttribute{
				Description: "Which environment this permission applies to.",
				Optional:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *permissionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read permission data source")
	var state permissionDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Unable to read permission data source")
		return
	}

	permissions, api_response, err := d.client.AuthAPI.GetPermissions(ctx).Execute()
	if (!ExpectedResponse(api_response, 200, &resp.Diagnostics, err)) {
		return
	}

	var permission *unleash.AdminPermissionSchema
	needle := state.Name.ValueString()
	if state.Environment.IsNull() {
		tflog.Debug(ctx, fmt.Sprintf("Provided empty environment, searching for %s", needle))
		for _, p := range permissions.Permissions.Root {
			if p.Name == needle {
				tflog.Debug(ctx, fmt.Sprintf("FOUND root %s as %v", needle, p))
				found := p
				permission = &found
				break
			}
		}

		if permission == nil {
			tflog.Debug(ctx, fmt.Sprintf("Did not find %s in root", needle))
			for _, p := range permissions.Permissions.Project {
				if p.Name == needle {
					tflog.Debug(ctx, fmt.Sprintf("FOUND project %s", needle))
					found := p
					permission = &found
					break
				}
			}
		}
	} else {
		for _, env := range permissions.Permissions.Environments {
			envName := state.Environment.ValueString()
			if env.Name == envName {
				tflog.Debug(ctx, fmt.Sprintf("Checking in env %s for %s", env.Name, needle))
				for _, p := range env.Permissions {
					if p.Name == needle {
						tflog.Debug(ctx, fmt.Sprintf("FOUND env %s: %v", needle, p))
						found := p
						permission = &found
						break
					}
				}
			}
		}
	}
	if permission == nil {
		resp.Diagnostics.AddError(
			"Permission not found",
			fmt.Sprintf("Permission %s not found", needle),
		)
		return
	}

	// Map response body to model
	state = permissionDataSourceModel{
		Id:          types.Int64Value(int64(permission.Id)),
		Name:        types.StringValue(permission.Name),
		Type:        types.StringValue(permission.Type),
		DisplayName: types.StringValue(permission.DisplayName),
	}
	if permission.Environment != nil {
		state.Environment = types.StringValue(*permission.Environment)
	} else {
		state.Environment = types.StringNull()
	}

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading permission data source", map[string]any{"success": true})
}
