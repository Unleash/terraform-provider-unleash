package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = &serviceAccountDataSource{}
	// _ datasource.DataSourceWithConfigure = &serviceAccountDataSource{}
)

func NewServiceAccountDataSource() datasource.DataSource {
	return &serviceAccountDataSource{}
}

type serviceAccountDataSource struct {
	client *unleash.APIClient
}

type serviceAccountModel struct {
	Id       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	UserName types.String `tfsdk:"type"`
	RootRole types.Int64  `tfsdk:"description"`
}

func (d *serviceAccountDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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

func (d *serviceAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *serviceAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a service account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the service account.",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username for the service account.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the service account.",
				Required:    true,
			},
			"root_role": schema.Int64Attribute{
				Description: "The root role ID for the service account.",
				Required:    true,
			},
		},
	}
}

func (d *serviceAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read service account data source")
	var state serviceAccountModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	serviceAccounts, api_response, err := d.client.ServiceAccountsAPI.GetServiceAccounts(ctx).Execute()
	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}
	var serviceAccount *unleash.ServiceAccountSchema

	for _, s := range serviceAccounts.ServiceAccounts {
		if fmt.Sprintf("%v", s.Id) == state.Id.ValueString() {
			serviceAccount = &s
			break
		}
	}

	if serviceAccount == nil {
		resp.Diagnostics.AddError("Error reading service account", "Could not find service account")
		return
	}

	state = serviceAccountModel{
		Id:       types.StringValue(fmt.Sprintf("%v", serviceAccount.Id)),
		Name:     types.StringValue(*serviceAccount.Name),
		UserName: types.StringValue(*serviceAccount.Username),
		RootRole: types.Int64Value(int64(*serviceAccount.RootRole)),
	}

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading service account data source", map[string]any{"success": true})
}

func (r *serviceAccountDataSource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var state serviceAccountModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createSchema := unleash.CreateServiceAccountSchema{
		Username: state.UserName.ValueStringPointer(),
		Name:     state.Name.ValueStringPointer(),
		RootRole: int32(state.RootRole.ValueInt64()),
	}

	serviceAccount, apiResponse, err := r.client.ServiceAccountsAPI.CreateServiceAccount(ctx).
		CreateServiceAccountSchema(createSchema).
		Execute()

	if !ValidateApiResponse(apiResponse, 201, &resp.Diagnostics, err) {
		return
	}

	state.Id = types.StringValue(fmt.Sprintf("%v", serviceAccount.Id))
	state.Name = types.StringValue(*serviceAccount.Name)
	state.UserName = types.StringValue(*serviceAccount.Username)
	state.RootRole = types.Int64Value(int64(*serviceAccount.RootRole))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished creating service account data source", map[string]any{"success": true})
}

func (r *serviceAccountDataSource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceAccountModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	api_response, err := r.client.ServiceAccountsAPI.DeleteServiceAccount(ctx, state.Id.ValueString()).Execute()

	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, "Deleted item resource", map[string]any{"success": true})
}
