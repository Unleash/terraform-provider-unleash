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

var (
	_ datasource.DataSource              = &roleDataSource{}
	_ datasource.DataSourceWithConfigure = &roleDataSource{}
)

func NewRoleDataSource() datasource.DataSource {
	return &roleDataSource{}
}

type roleDataSource struct {
	client *unleash.APIClient
}

type roleDataSourceModel struct {
	Id          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
}

func (d *roleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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

func (d *roleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *roleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a role definition.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of this role.",
				Required:    true,
			},
			"id": schema.Int64Attribute{
				Description: "The id of this role.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "A role can either be a global root role (applies to all roles) or a role role.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "A more detailed description of the role and what use it's intended for.",
				Computed:    true,
			},
		},
	}
}

func (d *roleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read role data source")
	var state roleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	roles, api_response, err := d.client.UsersAPI.GetRoles(ctx).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read roles",
			err.Error(),
		)
		return
	}

	if api_response.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"Unexpected HTTP error code received",
			api_response.Status,
		)
		return
	}

	var role unleash.RoleSchema
	for _, r := range roles.Roles {
		if r.Name == state.Name.ValueString() {
			role = r
		}
	}

	state = roleDataSourceModel{
		Id:   types.Int64Value(int64(role.Id)),
		Name: types.StringValue(fmt.Sprintf("%v", role.Name)),
		Type: types.StringValue(fmt.Sprintf("%v", role.Type)),
	}

	if role.Description != nil {
		state.Description = types.StringValue(fmt.Sprintf("%v", role.Description))
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading user data source", map[string]any{"success": true})
}
