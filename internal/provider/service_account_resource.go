package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &serviceAccountResource{}
	_ resource.ResourceWithConfigure   = &serviceAccountResource{}
	_ resource.ResourceWithImportState = &serviceAccountResource{}
)

func NewServiceAccountResource() resource.Resource {
	return &serviceAccountResource{}
}

type serviceAccountResource struct {
	client *unleash.APIClient
}

type serviceAccountResourceModel struct {
	Id       types.Int64  `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	UserName types.String `tfsdk:"username"`
	RootRole types.Int64  `tfsdk:"root_role"`
}

func (r *serviceAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	r.client = client
}

func (r *serviceAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account"
}

func (r *serviceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a service account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
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

func (r *serviceAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to read service account resource")

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Debug(ctx, "Finished reading service account resource")
}

func (r *serviceAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create service account")
	var plan serviceAccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	roleId, diags := plan.RootRole.ToInt64Value(ctx)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	plan.RootRole.ToInt64Value(ctx)

	roleId32 := int32(roleId.ValueInt64())

	createSchema := unleash.CreateServiceAccountSchema{
		Username: plan.UserName.ValueStringPointer(),
		Name:     plan.Name.ValueStringPointer(),
		RootRole: roleId32,
	}

	serviceAccount, apiResponse, err := r.client.ServiceAccountsAPI.CreateServiceAccount(ctx).
		CreateServiceAccountSchema(createSchema).
		Execute()

	if !ValidateApiResponse(apiResponse, 201, &resp.Diagnostics, err) {
		return
	}

	if serviceAccount.Name != nil {
		plan.Name = types.StringValue(*serviceAccount.Name)
	} else {
		plan.Name = types.StringNull()
	}

	if serviceAccount.Username != nil {
		plan.UserName = types.StringValue(*serviceAccount.Username)
	} else {
		plan.UserName = types.StringNull()
	}

	plan.RootRole = types.Int64Value(int64(*serviceAccount.RootRole))
	plan.Id = types.Int64Value(int64(serviceAccount.Id))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished creating service account resource", map[string]any{"success": true})
}

func (r *serviceAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read service account")
	var state serviceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccounts, apiResponse, err := r.client.ServiceAccountsAPI.GetServiceAccounts(ctx).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	var serviceAccount *unleash.ServiceAccountSchema

	for i := range serviceAccounts.ServiceAccounts {
		if fmt.Sprintf("%g", serviceAccounts.ServiceAccounts[i].Id) == state.Id.String() {
			serviceAccount = &serviceAccounts.ServiceAccounts[i]
			break
		}
	}

	if serviceAccount == nil {
		resp.Diagnostics.AddError("Service account not found", "no service account found with the given ID")
		return
	}

	if serviceAccount.Name != nil {
		state.Name = types.StringValue(*serviceAccount.Name)
	} else {
		state.Name = types.StringNull()
	}

	if serviceAccount.Username != nil {
		state.UserName = types.StringValue(*serviceAccount.Username)
	} else {
		state.UserName = types.StringNull()
	}

	state.RootRole = types.Int64Value(int64(*serviceAccount.RootRole))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading service account resource", map[string]any{"success": true})
}

func (r *serviceAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update service account")
	var state serviceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newRootRole := int32(state.RootRole.ValueInt64())

	updateSchema := unleash.UpdateServiceAccountSchema{
		RootRole: &newRootRole,
		Name:     state.Name.ValueStringPointer(),
	}

	accountId := fmt.Sprintf("%v", state.Id.ValueInt64())
	serviceAccount, apiResponse, err := r.client.ServiceAccountsAPI.UpdateServiceAccount(ctx, accountId).UpdateServiceAccountSchema(updateSchema).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	if serviceAccount.Name != nil {
		state.Name = types.StringValue(*serviceAccount.Name)
	} else {
		state.Name = types.StringNull()
	}

	if serviceAccount.Username != nil {
		state.UserName = types.StringValue(*serviceAccount.Username)
	} else {
		state.UserName = types.StringNull()
	}

	state.RootRole = types.Int64Value(int64(*serviceAccount.RootRole))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished updating service account resource", map[string]any{"success": true})
}

func (r *serviceAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete service account")
	var state serviceAccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	accountId := fmt.Sprintf("%v", state.Id.ValueInt64())
	apiResponse, err := r.client.ServiceAccountsAPI.DeleteServiceAccount(ctx, accountId).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	tflog.Debug(ctx, "Finished deleting service account", map[string]any{"success": true})
}
