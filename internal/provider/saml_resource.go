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
	_ resource.Resource              = &samlResource{}
	_ resource.ResourceWithConfigure = &samlResource{}
)

func NewSamlResource() resource.Resource {
	return &samlResource{}
}

type samlResource struct {
	client *client.APIClient
}

type samlResourceModel struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	Certificate     types.String `tfsdk:"certificate"`
	EntityId        types.String `tfsdk:"entity_id"`
	SignOnUrl       types.String `tfsdk:"sign_on_url"`
	AutoCreate      types.Bool   `tfsdk:"auto_create"`
	DefaultRootRole types.Int64  `tfsdk:"default_root_role"`
}

func (r *samlResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.APIClient)
	if !ok {
		return
	}
	r.client = client
}

func (r *samlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml"
}

func (r *samlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages SAML configuration.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether SAML is enabled.",
				Required:    true,
			},
			"certificate": schema.StringAttribute{
				Description: "The x509 certificate used by the SAML provider.",
				Required:    true,
			},
			"entity_id": schema.StringAttribute{
				Description: "The SAML entity ID.",
				Required:    true,
			},
			"sign_on_url": schema.StringAttribute{
				Description: "The SAML sign-on URL.",
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

func (r *samlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read SAML configuration")
	var plan samlResourceModel

	samlSettings, httpRes, err := r.client.AuthAPI.GetSamlSettings(ctx).Execute()

	if !ValidateApiResponse(httpRes, 200, &resp.Diagnostics, err) {
		return
	}

	if !samlSettings.GetEnabled() {
		tflog.Warn(ctx, "SAML is not enabled, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	plan.Enabled = types.BoolValue(samlSettings.GetEnabled())
	plan.Certificate = types.StringValue(samlSettings.GetCertificate())
	plan.EntityId = types.StringValue(samlSettings.GetEntityId())
	plan.SignOnUrl = types.StringValue(samlSettings.GetSignOnUrl())
	plan.DefaultRootRole = types.Int64Value(int64(samlSettings.GetDefaultRootRoleId()))
	plan.AutoCreate = types.BoolValue(samlSettings.GetAutoCreate())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished reading SAML configuration")
}

func (r *samlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create SAML configuration")
	var plan samlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	samlSettingsResponse, err := updateSamlConfig(ctx, plan, r.client, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAML configuration", err.Error())
		return
	}

	plan.AutoCreate = types.BoolValue(samlSettingsResponse.GetAutoCreate())
	plan.Certificate = types.StringValue(samlSettingsResponse.GetCertificate())
	plan.DefaultRootRole = types.Int64Value(int64(samlSettingsResponse.GetDefaultRootRoleId()))
	plan.EntityId = types.StringValue(samlSettingsResponse.GetEntityId())
	plan.Enabled = types.BoolValue(samlSettingsResponse.GetEnabled())
	plan.SignOnUrl = types.StringValue(samlSettingsResponse.GetSignOnUrl())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished creating SAML configuration")
}

func (r *samlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update SAML configuration")
	var plan samlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	samlSettingsResponse, err := updateSamlConfig(ctx, plan, r.client, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAML configuration", err.Error())
		return
	}

	plan.AutoCreate = types.BoolValue(samlSettingsResponse.GetAutoCreate())
	plan.Certificate = types.StringValue(samlSettingsResponse.GetCertificate())
	plan.DefaultRootRole = types.Int64Value(int64(samlSettingsResponse.GetDefaultRootRoleId()))
	plan.EntityId = types.StringValue(samlSettingsResponse.GetEntityId())
	plan.Enabled = types.BoolValue(samlSettingsResponse.GetEnabled())
	plan.SignOnUrl = types.StringValue(samlSettingsResponse.GetSignOnUrl())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished updating SAML configuration")
}

func (r *samlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func updateSamlConfig(ctx context.Context, plan samlResourceModel, apiClient *client.APIClient, diagnostics *diag.Diagnostics) (*client.SamlSettingsResponseSchema, error) {
	preSamlSettings, preHttpRes, preErr := apiClient.AuthAPI.GetSamlSettings(ctx).Execute()

	if !ValidateApiResponse(preHttpRes, 200, diagnostics, preErr) {
		return nil, preErr
	}

	innerSettings := client.NewSamlSettingsSchemaOneOfWithDefaults()
	innerSettings.SetEnabled(plan.Enabled.ValueBool())

	if !plan.Certificate.IsNull() {
		innerSettings.SetCertificate(plan.Certificate.ValueString())
	} else {
		innerSettings.SetCertificate(preSamlSettings.GetCertificate())
	}

	if !plan.EntityId.IsNull() {
		innerSettings.SetEntityId(plan.EntityId.ValueString())
	} else {
		innerSettings.SetEntityId(preSamlSettings.GetEntityId())
	}

	if !plan.SignOnUrl.IsNull() {
		innerSettings.SetSignOnUrl(plan.SignOnUrl.ValueString())
	} else {
		innerSettings.SetSignOnUrl(preSamlSettings.GetSignOnUrl())
	}

	if !plan.DefaultRootRole.IsNull() {
		defaultRootRole := float32(plan.DefaultRootRole.ValueInt64())
		innerSettings.SetDefaultRootRoleId(defaultRootRole)
	} else {
		innerSettings.SetDefaultRootRoleId(preSamlSettings.GetDefaultRootRoleId())
	}

	if !plan.AutoCreate.IsNull() {
		innerSettings.SetAutoCreate(plan.AutoCreate.ValueBool())
	} else {
		innerSettings.SetAutoCreate(preSamlSettings.GetAutoCreate())
	}

	samlSettings := client.SamlSettingsSchema{
		SamlSettingsSchemaOneOf: innerSettings,
	}

	_, httpRes, err := apiClient.AuthAPI.SetSamlSettings(ctx).SamlSettingsSchema(samlSettings).Execute()

	if !ValidateApiResponse(httpRes, 200, diagnostics, err) {
		return nil, err
	}

	// Gotta do a second get because the response from the post is missing some fields
	samlSettingsResponse, httpRes, err := apiClient.AuthAPI.GetSamlSettings(ctx).Execute()

	if !ValidateApiResponse(httpRes, 200, diagnostics, err) {
		return nil, err
	}

	return samlSettingsResponse, nil
}
