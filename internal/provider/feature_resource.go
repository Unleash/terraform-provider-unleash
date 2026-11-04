package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &featureResource{}
	_ resource.ResourceWithConfigure   = &featureResource{}
	_ resource.ResourceWithImportState = &featureResource{}
)

// Unleash requires flag names to be URL-friendly.
var featureNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\-._~]+$`)

const defaultFeatureProject = "default"

func NewFeatureResource() resource.Resource {
	return &featureResource{}
}

type featureResource struct {
	client *unleash.APIClient
}

type featureResourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Project          types.String `tfsdk:"project"`
	Type             types.String `tfsdk:"type"`
	Description      types.String `tfsdk:"description"`
	ImpressionData   types.Bool   `tfsdk:"impression_data"`
	ArchiveOnDestroy types.Bool   `tfsdk:"archive_on_destroy"`
}

func (r *featureResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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

func (r *featureResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature"
}

func (r *featureResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a feature flag and manages its metadata (type, description and impression data). " +
			"Strategies, variants and whether the flag is enabled in each environment are intentionally not managed here: " +
			"change those in Unleash.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The name of the feature flag.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the feature flag. Must be unique across the instance, including archived flags, " +
					"and may only contain letters, numbers and the characters `-._~`. Changing it creates a new flag.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(featureNameRegex, "must only contain letters, numbers and the characters -._~"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project": schema.StringAttribute{
				Description: "The project the feature flag belongs to. Defaults to `default`. Changing it creates a new flag.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultFeatureProject),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The [type](https://docs.getunleash.io/concepts/feature-flags#feature-flag-types) of the feature flag, " +
					"for example `release`, `experiment`, `operational`, `kill-switch` or `permission`. Defaults to `release`.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("release"),
			},
			"description": schema.StringAttribute{
				Description: "A description of the feature flag.",
				Optional:    true,
			},
			"impression_data": schema.BoolAttribute{
				Description: "Whether to emit [impression data](https://docs.getunleash.io/concepts/impression-data) for this flag. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"archive_on_destroy": schema.BoolAttribute{
				Description: "When `true` (the default), destroying this resource archives the flag, as deleting a flag in the Unleash UI does, " +
					"so it can be revived later. When `false`, the flag is also permanently deleted from the archive, which frees up its name.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
		},
	}
}

// parseFeatureImportId accepts "<project>/<name>", or just "<name>" for a flag in the default project.
func parseFeatureImportId(id string) (project string, name string, err error) {
	parts := strings.Split(id, "/")
	switch {
	case len(parts) == 1 && parts[0] != "":
		return defaultFeatureProject, parts[0], nil
	case len(parts) == 2 && parts[0] != "" && parts[1] != "":
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("expected an import ID of the form <project>/<name> or <name>, got %q", id)
	}
}

func (r *featureResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	project, name, err := parseFeatureImportId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	// not stored in Unleash, so an imported flag gets the default
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("archive_on_destroy"), true)...)
}

func (m *featureResourceModel) hydrateFromApi(feature featureSchema) {
	m.Id = types.StringValue(feature.Name)
	m.Name = types.StringValue(feature.Name)
	m.Project = types.StringValue(feature.Project)
	m.Type = types.StringValue(feature.Type)
	m.ImpressionData = types.BoolValue(feature.ImpressionData)
	// Unleash stores a removed description as an empty string; keep it null so plans stay clean
	if feature.Description == nil || *feature.Description == "" {
		m.Description = types.StringNull()
	} else {
		m.Description = types.StringValue(*feature.Description)
	}
}

func (r *featureResource) api() featureAPI {
	return featureAPI{client: r.client}
}

func (r *featureResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create feature resource")
	var plan featureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpRes, err := r.api().create(ctx, plan.Project.ValueString(), createFeatureRequest{
		Name:           plan.Name.ValueString(),
		Type:           plan.Type.ValueString(),
		Description:    plan.Description.ValueString(),
		ImpressionData: plan.ImpressionData.ValueBool(),
	})
	if !ValidateApiResponse(httpRes, http.StatusCreated, &resp.Diagnostics, err) {
		return
	}

	feature, httpRes, err := r.api().get(ctx, plan.Project.ValueString(), plan.Name.ValueString())
	if !ValidateApiResponse(httpRes, http.StatusOK, &resp.Diagnostics, err) {
		return
	}

	plan.hydrateFromApi(*feature)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished creating feature resource", map[string]interface{}{"success": true})
}

func (r *featureResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read feature resource")
	var state featureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// an archived flag is a 404 here, so it is removed from state and recreated on the next apply
	feature, httpRes, err := r.api().get(ctx, state.Project.ValueString(), state.Name.ValueString())
	if !ValidateReadApiResponse(ctx, httpRes, err, resp, state.Name.ValueString(), "Feature flag") {
		return
	}

	state.hydrateFromApi(*feature)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading feature resource", map[string]interface{}{"success": true})
}

func (r *featureResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update feature resource")
	var plan featureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpRes, err := r.api().update(ctx, plan.Project.ValueString(), plan.Name.ValueString(), updateFeatureRequest{
		Type:           plan.Type.ValueString(),
		Description:    plan.Description.ValueString(),
		ImpressionData: plan.ImpressionData.ValueBool(),
	})
	if !ValidateApiResponse(httpRes, http.StatusOK, &resp.Diagnostics, err) {
		return
	}

	feature, httpRes, err := r.api().get(ctx, plan.Project.ValueString(), plan.Name.ValueString())
	if !ValidateApiResponse(httpRes, http.StatusOK, &resp.Diagnostics, err) {
		return
	}

	plan.hydrateFromApi(*feature)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished updating feature resource", map[string]interface{}{"success": true})
}

func (r *featureResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete feature resource")
	var state featureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpRes, err := r.api().archive(ctx, state.Project.ValueString(), state.Name.ValueString())
	// 404: already archived or deleted outside Terraform
	if !IsValidApiResponse(httpRes, []int{http.StatusAccepted, http.StatusOK, http.StatusNotFound}, &resp.Diagnostics, err) {
		return
	}

	if !state.ArchiveOnDestroy.IsNull() && !state.ArchiveOnDestroy.ValueBool() {
		httpRes, err = r.api().deleteArchived(ctx, state.Name.ValueString())
		if !IsValidApiResponse(httpRes, []int{http.StatusOK, http.StatusNoContent, http.StatusNotFound}, &resp.Diagnostics, err) {
			return
		}
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, "Finished deleting feature resource", map[string]interface{}{"success": true})
}
