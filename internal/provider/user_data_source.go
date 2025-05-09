package provider

import (
	"context"
	"fmt"
	"strconv"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

// NewUserDataSource is a helper function to simplify the provider implementation.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource is the data source implementation.
type userDataSource struct {
	client *unleash.APIClient
}

type userDataSourceModel struct {
	Id       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Email    types.String `tfsdk:"email"`
	Name     types.String `tfsdk:"name"`
	RootRole types.Int64  `tfsdk:"root_role"`
}

// Configure adds the provider configured client to the data source.
func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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
func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the data source. TODO: can we transform OpenAPI schema into TF schema?
func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this user.",
				Required:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email of the user.",
				Computed:    true,
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username of the user.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the user.",
				Optional:    true,
				Computed:    true,
			},
			"root_role": schema.Int64Attribute{
				Description: "The role id for the user.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read user data source")
	var state userDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	userId, err := strconv.Atoi(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("User id was not a number %s", state.Id.ValueString()),
			err.Error(),
		)
		return
	}

	user, api_response, err := d.client.UsersAPI.GetUser(ctx, int32(userId)).Execute()
	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	// Map response body to model
	state = userDataSourceModel{
		Id:       types.StringValue(fmt.Sprintf("%v", user.Id)),
		RootRole: types.Int64Value(int64(*user.RootRole)),
	}
	if user.Username.IsSet() {
		state.Username = types.StringValue(*user.Username.Get())
	} else {
		state.Username = types.StringNull()
	}
	if user.Email != nil {
		state.Email = types.StringValue(*user.Email)
	} else {
		state.Email = types.StringNull()
	}
	if user.Name.IsSet() {
		state.Name = types.StringValue(*user.Name.Get())
	} else {
		state.Name = types.StringNull()
	}

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading user data source", map[string]any{"success": true})
}
