package provider

import (
	"context"
	"fmt"
	"time"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &serviceAccountTokensResource{}
	_ resource.ResourceWithConfigure   = &serviceAccountTokensResource{}
	_ resource.ResourceWithImportState = &serviceAccountTokensResource{}
)

func NewServiceAccountTokensResource() resource.Resource {
	return &serviceAccountTokensResource{}
}

type serviceAccountTokensResource struct {
	client *unleash.APIClient
}

type serviceAccountTokensResourceModel struct {
	Id               types.Int64  `tfsdk:"id"`
	ServiceAccountId types.Int64  `tfsdk:"service_account_id"`
	Secret           types.String `tfsdk:"secret"`
	Description      types.String `tfsdk:"description"`
	ExpiresAt        types.String `tfsdk:"expires_at"`
}

func (r *serviceAccountTokensResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		return
	}
	r.client = client
}

func (r *serviceAccountTokensResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_account_token"
}

func (r *serviceAccountTokensResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages service account tokens.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The ID of the service account.",
				Computed:    true,
			},
			"service_account_id": schema.Int64Attribute{
				Description: "The ID of the service account token.",
				Required:    true,
			},
			"secret": schema.StringAttribute{
				Description: "The secret of the service account token.",
				Sensitive:   true,
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the service account token.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"expires_at": schema.StringAttribute{
				Description: "The expiration date of the service account token.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *serviceAccountTokensResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to read service account tokens resource")
	resource.ImportStatePassthroughID(ctx, path.Root("service_account"), req, resp)
	tflog.Debug(ctx, "Finished importing service account tokens data source", map[string]interface{}{"success": true})
}

func (r *serviceAccountTokensResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create service account tokens")
	var state serviceAccountTokensResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	expiresAt, err := time.Parse(time.RFC3339, state.ExpiresAt.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create service account token",
			fmt.Sprintf("Failed to parse expiration date: %v", err),
		)
		return
	}

	createSchema := unleash.CreatePatSchema{
		Description: state.Description.ValueString(),
		ExpiresAt:   expiresAt,
	}

	serviceAccountId := fmt.Sprintf("%v", state.ServiceAccountId.ValueInt64())

	serviceAccountToken, apiResponse, err := r.client.ServiceAccountsAPI.CreateServiceAccountToken(ctx, serviceAccountId).CreatePatSchema(createSchema).Execute()

	if !ValidateApiResponse(apiResponse, 201, &resp.Diagnostics, err) {
		return
	}

	state.ExpiresAt = types.StringValue(serviceAccountToken.ExpiresAt.Format(time.RFC3339))
	state.Id = types.Int64Value(int64(serviceAccountToken.Id))
	state.Description = types.StringValue(serviceAccountToken.Description)

	if serviceAccountToken.Secret != nil {
		state.Secret = types.StringValue(*serviceAccountToken.Secret)
	} else {
		// not sure why our API thinks this can be returned as null
		// but I'm pretty sure that can't/shouldn't happen and if it does
		// then the this is very broken, better to error out than to continue
		resp.Diagnostics.AddError(
			"Failed to create service account token",
			"Secret was null when token was created, token is not valid",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished creating service account tokens resource", map[string]interface{}{"success": true})
}

func (r *serviceAccountTokensResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read service account tokens resource")
	var state serviceAccountTokensResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccountId := fmt.Sprintf("%v", state.ServiceAccountId.ValueInt64())

	serviceAccountTokens, apiResponse, err := r.client.ServiceAccountsAPI.GetServiceAccountTokens(ctx, serviceAccountId).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	var serviceAccountToken *unleash.PatSchema

	for _, s := range serviceAccountTokens.Pats {
		if s.Id == int32(state.Id.ValueInt64()) {
			serviceAccountToken = &s
			break
		}
	}

	if serviceAccountToken == nil {
		resp.Diagnostics.AddError("Service account token not found", "no service account token found with the given ID")
		return
	}

	state.ExpiresAt = types.StringValue(serviceAccountToken.ExpiresAt.Format(time.RFC3339))
	state.Description = types.StringValue(serviceAccountToken.Description)
	state.Id = types.Int64Value(int64(serviceAccountToken.Id))
	state.Secret = types.StringNull() //Can't get the token again, it's only made during creation, so let terraform know it's gone now

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading service account tokens data source", map[string]interface{}{"success": true})
}

func (r *serviceAccountTokensResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	//There's no update in the API and it _really_ doesn't make sense anyway
	resp.Diagnostics.AddError("Service account tokens do not support updates", "Service account tokens are immutable")
}

func (r *serviceAccountTokensResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete service account tokens resource")
	var state serviceAccountTokensResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccountId := fmt.Sprintf("%v", state.ServiceAccountId.ValueInt64())
	serviceAccountTokenId := fmt.Sprintf("%v", state.Id.ValueInt64())

	apiResponse, err := r.client.ServiceAccountsAPI.DeleteServiceAccountToken(ctx, serviceAccountId, serviceAccountTokenId).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished deleting service account tokens resource", map[string]interface{}{"success": true})
}
