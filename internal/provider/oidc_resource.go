package provider

import (
	"context"

	"github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &oidcResource{}
	_ resource.ResourceWithConfigure = &oidcResource{}
)

func NewOidcResource() resource.Resource {
	return &oidcResource{}
}

type oidcResource struct {
	client *client.APIClient
}

type oidcResourceModel struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	DefaultRootRole types.Int64  `tfsdk:"default_root_role"`
	DiscoverUrl     types.String `tfsdk:"discover_url"`
	ClientId        types.String `tfsdk:"client_id"`
	Secret          types.String `tfsdk:"secret"`
	AutoCreate      types.Bool   `tfsdk:"auto_create"`
}

func (r *oidcResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.APIClient)
	if !ok {
		return
	}
	r.client = client
}

func (r *oidcResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc"
}

func (r *oidcResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages OIDC configuration.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether or not OIDC is enabled.",
				Required:    true,
			},
			"discover_url": schema.StringAttribute{
				Description: "A URL pointing to the .well-known configuration of the OIDC provider.",
				Required:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "The OIDC public identifier.",
				Required:    true,
			},
			"secret": schema.StringAttribute{
				Description: "The OIDC secret.",
				Required:    true,
			},
			"auto_create": schema.BoolAttribute{
				Description: "Whether to auto create users when they login to Unleash for the first time.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"default_root_role": schema.Int64Attribute{
				Description: "The default root role give to a user when that user is created. Only used if auto_create is set to true.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *oidcResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read OIDC configuration")
	var plan oidcResourceModel

	oidcSettings, httpRes, err := r.client.AuthAPI.GetOidcSettings(context.Background()).Execute()

	if !ValidateApiResponse(httpRes, 200, &resp.Diagnostics, err) {
		return
	}

	plan.ClientId = types.StringValue(oidcSettings.GetClientId())
	plan.DiscoverUrl = types.StringValue(oidcSettings.GetDiscoverUrl())
	plan.Enabled = types.BoolValue(oidcSettings.GetEnabled())
	plan.Secret = types.StringValue(oidcSettings.GetSecret())

	plan.AutoCreate = types.BoolValue(oidcSettings.GetAutoCreate())
	plan.DefaultRootRole = types.Int64Value(int64(oidcSettings.GetDefaultRootRoleId()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "OIDC configuration read")
}

func (r *oidcResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create OIDC configuration")
	var plan oidcResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oidcSettingsResponse, err := updateOidcConfig(plan, r.client, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update OIDC configuration", err.Error())
		return
	}

	plan.ClientId = types.StringValue(oidcSettingsResponse.GetClientId())
	plan.DiscoverUrl = types.StringValue(oidcSettingsResponse.GetDiscoverUrl())
	plan.Enabled = types.BoolValue(oidcSettingsResponse.GetEnabled())
	plan.Secret = types.StringValue(oidcSettingsResponse.GetSecret())
	plan.AutoCreate = types.BoolValue(oidcSettingsResponse.GetAutoCreate())
	plan.DefaultRootRole = types.Int64Value(int64(oidcSettingsResponse.GetDefaultRootRoleId()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "OIDC configuration created")
}

func (r *oidcResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update OIDC configuration")
	var plan oidcResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	oidcSettingsResponse, err := updateOidcConfig(plan, r.client, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update OIDC configuration", err.Error())
		return
	}

	plan.ClientId = types.StringValue(oidcSettingsResponse.GetClientId())
	plan.DiscoverUrl = types.StringValue(oidcSettingsResponse.GetDiscoverUrl())
	plan.Enabled = types.BoolValue(oidcSettingsResponse.GetEnabled())
	plan.Secret = types.StringValue(oidcSettingsResponse.GetSecret())
	plan.AutoCreate = types.BoolValue(oidcSettingsResponse.GetAutoCreate())
	plan.DefaultRootRole = types.Int64Value(int64(oidcSettingsResponse.GetDefaultRootRoleId()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "OIDC configuration updated")
}

func (r *oidcResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to remove OIDC configuration")
	var plan oidcResourceModel

	if resp.Diagnostics.HasError() {
		return
	}

	innerSettings := client.NewOidcSettingsSchemaOneOfWithDefaults()
	innerSettings.SetEnabled(false)
	// These two properties must exist on Unleash versions prior to 6.0 but can't be empty strings
	// This is the same thing the frontend does when it clears this
	innerSettings.SetClientId(" ")
	innerSettings.SetSecret(" ")

	oidcSettings := client.OidcSettingsSchema{
		OidcSettingsSchemaOneOf: innerSettings,
	}

	_, httpRes, err := r.client.AuthAPI.SetOidcSettings(context.Background()).OidcSettingsSchema(oidcSettings).Execute()

	if !ValidateApiResponse(httpRes, 200, &resp.Diagnostics, err) {
		return
	}

	resp.State.RemoveResource(ctx)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "OIDC configuration cleared")
}

func updateOidcConfig(plan oidcResourceModel, apiClient *client.APIClient, diagnostics *diag.Diagnostics) (*client.OidcSettingsResponseSchema, error) {

	preOidcSettings, preHttpRes, preErr := apiClient.AuthAPI.GetOidcSettings(context.Background()).Execute()

	if !ValidateApiResponse(preHttpRes, 200, diagnostics, preErr) {
		return nil, preErr
	}

	innerSettings := client.NewOidcSettingsSchemaOneOfWithDefaults()
	innerSettings.SetEnabled(plan.Enabled.ValueBool())

	if !plan.ClientId.IsNull() {
		innerSettings.SetClientId(plan.ClientId.ValueString())
	} else {
		innerSettings.SetClientId(preOidcSettings.GetClientId())
	}

	if !plan.Secret.IsNull() {
		innerSettings.SetSecret(plan.Secret.ValueString())
	} else {
		innerSettings.SetSecret(preOidcSettings.GetSecret())
	}

	if !plan.DiscoverUrl.IsNull() {
		innerSettings.SetDiscoverUrl(plan.DiscoverUrl.ValueString())
	} else {
		innerSettings.SetDiscoverUrl(preOidcSettings.GetDiscoverUrl())
	}

	if !plan.AutoCreate.IsNull() {
		innerSettings.SetAutoCreate(plan.AutoCreate.ValueBool())
	} else {
		innerSettings.SetAutoCreate(preOidcSettings.GetAutoCreate())
	}

	if !plan.DefaultRootRole.IsNull() {
		defaultRootRole := float32(plan.DefaultRootRole.ValueInt64())
		innerSettings.SetDefaultRootRoleId(defaultRootRole)
	} else {
		innerSettings.SetDefaultRootRoleId(preOidcSettings.GetDefaultRootRoleId())
	}

	oidcSettings := client.OidcSettingsSchema{
		OidcSettingsSchemaOneOf: innerSettings,
	}

	_, httpRes, err := apiClient.AuthAPI.SetOidcSettings(context.Background()).OidcSettingsSchema(oidcSettings).Execute()

	if !ValidateApiResponse(httpRes, 200, diagnostics, err) {
		return nil, err
	}

	// the post request does not return everything, so we need to do another get to get the full object
	oidcSettingsResponse, httpRes, err := apiClient.AuthAPI.GetOidcSettings(context.Background()).Execute()

	if !ValidateApiResponse(httpRes, 200, diagnostics, err) {
		return nil, err
	}

	return oidcSettingsResponse, nil
}
