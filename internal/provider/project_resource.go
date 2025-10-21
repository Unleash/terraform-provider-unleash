package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *unleash.APIClient
}

type projectResourceModel struct {
	Id            types.String               `tfsdk:"id"`
	Name          types.String               `tfsdk:"name"`
	Description   types.String               `tfsdk:"description"`
	Mode          types.String               `tfsdk:"mode"`
	FeatureNaming *featureNamingModel        `tfsdk:"feature_naming"`
	LinkTemplates []projectLinkTemplateModel `tfsdk:"link_templates"`
}

type featureNamingModel struct {
	Pattern     types.String `tfsdk:"pattern"`
	Example     types.String `tfsdk:"example"`
	Description types.String `tfsdk:"description"`
}

type projectLinkTemplateModel struct {
	Title       types.String `tfsdk:"title"`
	UrlTemplate types.String `tfsdk:"url_template"`
}

// Configure adds the provider configured client to the data source.
func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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

// Metadata returns the data source type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the data source. TODO: can we transform OpenAPI schema into TF schema?
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Project schema",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this project.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the project.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the project's purpose.",
				Optional:    true,
			},
			"mode": schema.StringAttribute{
				Description: "The project's collaboration mode. Determines whether non project members can submit " +
					"change requests and the projects visibility to non members. Valid values are 'open', 'protected' and 'private'." +
					" If a value is not set, the project will default to 'open'",
				Computed: true,
				Optional: true,
			},
			"feature_naming": schema.SingleNestedAttribute{
				Description: "Optional feature naming pattern applied to all features created in this project.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"pattern": schema.StringAttribute{
						Description: "A JavaScript regular expression pattern, without the start and end delimiters.",
						Required:    true,
					},
					"example": schema.StringAttribute{
						Description: "An example feature name that matches the pattern.",
						Optional:    true,
					},
					"description": schema.StringAttribute{
						Description: "A human-readable description of the pattern.",
						Optional:    true,
					},
				},
			},
			"link_templates": schema.ListNestedAttribute{
				Description: "Optional list of link templates automatically added to new feature flags.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"title": schema.StringAttribute{
							Description: "Link title shown in the Unleash UI.",
							Optional:    true,
						},
						"url_template": schema.StringAttribute{
							Description: "URL template that can contain {{project}} or {{feature}} placeholders.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import project resource")

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Debug(ctx, "Finished importing project data source", map[string]any{"success": true})
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create project resource")
	var plan projectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createProjectRequest := *unleash.NewCreateProjectSchemaWithDefaults()
	createProjectRequest.Name = *plan.Name.ValueStringPointer()
	createProjectRequest.Id = plan.Id.ValueStringPointer()
	createProjectRequest.Environments = []string{}
	if !plan.Description.IsNull() {
		createProjectRequest.Description = *unleash.NewNullableString(plan.Description.ValueStringPointer())
	}

	project, api_response, err := r.client.ProjectsAPI.CreateProject(ctx).CreateProjectSchema(createProjectRequest).Execute()

	if !ValidateApiResponse(api_response, 201, &resp.Diagnostics, err) {
		return
	}

	mode, err := resolveRequestedMode(plan)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "InvalidMode")
		return
	}

	updateProjectSettingsRequest := *unleash.NewUpdateProjectEnterpriseSettingsSchemaWithDefaults()
	updateProjectSettingsRequest.SetMode(mode)

	if featureNaming, set := expandFeatureNaming(plan.FeatureNaming, &resp.Diagnostics); set {
		if resp.Diagnostics.HasError() {
			return
		}
		updateProjectSettingsRequest.SetFeatureNaming(*featureNaming)
	} else if resp.Diagnostics.HasError() {
		return
	}

	if linkTemplates, set := expandLinkTemplates(plan.LinkTemplates, &resp.Diagnostics); set {
		if resp.Diagnostics.HasError() {
			return
		}
		updateProjectSettingsRequest.SetLinkTemplates(linkTemplates)
	}

	updateSettingsResponse, err := r.client.ProjectsAPI.UpdateProjectEnterpriseSettings(ctx, *plan.Id.ValueStringPointer()).UpdateProjectEnterpriseSettingsSchema(updateProjectSettingsRequest).Execute()

	if !ValidateApiResponse(updateSettingsResponse, 200, &resp.Diagnostics, err) {
		return
	}

	plan.Id = types.StringValue(project.Id)
	plan.Name = types.StringValue(project.Name)
	plan.Mode = types.StringValue(mode)

	if project.Description.IsSet() {
		plan.Description = types.StringValue(*project.Description.Get())
	} else {
		plan.Description = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished creating project data source", map[string]any{"success": true})
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read project resource")
	var state projectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectId := state.Id.ValueString()
	projects, api_response, err := r.client.ProjectsAPI.GetProjects(ctx).Execute()

	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	tflog.Debug(ctx, "Searching for project", map[string]any{"id": projectId})
	var project *unleash.ProjectSchema
	for i := range projects.Projects {
		if projects.Projects[i].Id == projectId {
			project = &projects.Projects[i]
			break
		}
	}

	if project == nil {
		tflog.Warn(ctx, fmt.Sprintf("Project with id %s not found, removing from state", projectId))
		resp.State.RemoveResource(ctx)
		return
	}

	state.Id = types.StringValue(fmt.Sprintf("%v", project.Id))
	state.Name = types.StringValue(fmt.Sprintf("%v", project.Name))

	setModelMode(project.Mode, &state)

	if project.Description.IsSet() && project.Description.Get() != nil {
		state.Description = types.StringValue(*project.Description.Get())
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading project data source", map[string]any{"success": true})
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update project resource")
	var plan projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateProjectSchema := *unleash.NewUpdateProjectSchemaWithDefaults()
	updateProjectSchema.Name = *plan.Name.ValueStringPointer()
	if !plan.Description.IsNull() {
		updateProjectSchema.Description = plan.Description.ValueStringPointer()
	}

	if plan.Id.IsNull() || plan.Id.IsUnknown() {
		var state projectResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Id = state.Id
	}

	mode, err := resolveRequestedMode(plan)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "InvalidMode")
		return
	}

	updateProjectSettingsRequest := *unleash.NewUpdateProjectEnterpriseSettingsSchemaWithDefaults()
	updateProjectSettingsRequest.SetMode(mode)

	if featureNaming, set := expandFeatureNaming(plan.FeatureNaming, &resp.Diagnostics); set {
		if resp.Diagnostics.HasError() {
			return
		}
		updateProjectSettingsRequest.SetFeatureNaming(*featureNaming)
	} else if resp.Diagnostics.HasError() {
		return
	}

	if linkTemplates, set := expandLinkTemplates(plan.LinkTemplates, &resp.Diagnostics); set {
		if resp.Diagnostics.HasError() {
			return
		}
		updateProjectSettingsRequest.SetLinkTemplates(linkTemplates)
	}

	updateSettingsResponse, err := r.client.ProjectsAPI.UpdateProjectEnterpriseSettings(ctx, *plan.Id.ValueStringPointer()).UpdateProjectEnterpriseSettingsSchema(updateProjectSettingsRequest).Execute()

	if !ValidateApiResponse(updateSettingsResponse, 200, &resp.Diagnostics, err) {
		return
	}

	api_response, err := r.client.ProjectsAPI.UpdateProject(ctx, plan.Id.ValueString()).UpdateProjectSchema(updateProjectSchema).Execute()

	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	// our update doesn't return the project, so we need to re-read it
	projects, api_response, err := r.client.ProjectsAPI.GetProjects(ctx).Execute()

	var project unleash.ProjectSchema
	for _, p := range projects.Projects {
		if p.Id == plan.Id.ValueString() {
			project = p
		}
	}
	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	plan.Id = types.StringValue(fmt.Sprintf("%v", project.Id))
	plan.Name = types.StringValue(fmt.Sprintf("%v", project.Name))

	setModelMode(project.Mode, &plan)

	if project.Description.IsSet() {
		plan.Description = types.StringValue(*project.Description.Get())
	} else {
		plan.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished updating project data source", map[string]any{"success": true})
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete project")
	var state projectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	api_response, err := r.client.ProjectsAPI.DeleteProject(ctx, state.Id.ValueString()).Execute()

	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, "Deleted item resource", map[string]any{"success": true})
}

func setModelMode(mode *string, model *projectResourceModel) {
	if mode != nil {
		model.Mode = types.StringValue(*mode)
	} else {
		// From checking the API spec I don't believe this actually can happen but this gives us a nice
		// chance to have some backwards compatibility with older versions of the API where open was the only mode
		model.Mode = types.StringValue("open")
	}
}

func resolveRequestedMode(plan projectResourceModel) (string, error) {
	if !plan.Mode.IsNull() && plan.Mode.ValueString() != "" && plan.Mode.ValueString() != "open" && plan.Mode.ValueString() != "protected" && plan.Mode.ValueString() != "private" {
		return "", fmt.Errorf("project mode must be unset or set to 'open', 'protected' or 'private'. Got: '%s'", plan.Mode.ValueString())
	}

	if !plan.Mode.IsNull() && plan.Mode.ValueString() != "" {
		return plan.Mode.ValueString(), nil
	} else {
		return "open", nil
	}
}

func expandFeatureNaming(model *featureNamingModel, diagnostics *diag.Diagnostics) (*unleash.CreateFeatureNamingPatternSchema, bool) {
	if model == nil {
		return nil, false
	}

	if model.Pattern.IsUnknown() {
		diagnostics.AddError("Invalid feature_naming.pattern", "feature_naming.pattern cannot be unknown")
		return nil, false
	}

	if model.Pattern.IsNull() || model.Pattern.ValueString() == "" {
		diagnostics.AddError("Invalid feature_naming.pattern", "feature_naming.pattern must be provided and cannot be empty")
		return nil, false
	}

	featureNaming := unleash.CreateFeatureNamingPatternSchema{}
	featureNaming.SetPattern(model.Pattern.ValueString())

	if model.Example.IsUnknown() {
		diagnostics.AddError("Invalid feature_naming.example", "feature_naming.example cannot be unknown")
		return nil, false
	}

	if model.Example.IsNull() {
		featureNaming.SetExampleNil()
	} else {
		featureNaming.SetExample(model.Example.ValueString())
	}

	if model.Description.IsUnknown() {
		diagnostics.AddError("Invalid feature_naming.description", "feature_naming.description cannot be unknown")
		return nil, false
	}

	if model.Description.IsNull() {
		featureNaming.SetDescriptionNil()
	} else {
		featureNaming.SetDescription(model.Description.ValueString())
	}

	return &featureNaming, true
}

func expandLinkTemplates(models []projectLinkTemplateModel, diagnostics *diag.Diagnostics) ([]unleash.ProjectLinkTemplateSchema, bool) {
	if models == nil {
		return nil, false
	}

	templates := make([]unleash.ProjectLinkTemplateSchema, len(models))

	for i, model := range models {
		if model.UrlTemplate.IsUnknown() {
			diagnostics.AddError("Invalid link_templates url", fmt.Sprintf("link_templates[%d].url_template cannot be unknown", i))
			continue
		}

		if model.UrlTemplate.IsNull() || model.UrlTemplate.ValueString() == "" {
			diagnostics.AddError("Invalid link_templates url", fmt.Sprintf("link_templates[%d].url_template must be provided and cannot be empty", i))
			continue
		}

		template := unleash.NewProjectLinkTemplateSchema(model.UrlTemplate.ValueString())

		if model.Title.IsUnknown() {
			diagnostics.AddError("Invalid link_templates title", fmt.Sprintf("link_templates[%d].title cannot be unknown", i))
			continue
		}

		if model.Title.IsNull() {
			template.SetTitleNil()
		} else {
			template.SetTitle(model.Title.ValueString())
		}

		templates[i] = *template
	}

	return templates, true
}
