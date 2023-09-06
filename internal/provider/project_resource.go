package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
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
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
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
		},
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import project resource")
	var state projectResourceModel

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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
	createProjectRequest.Id = *plan.ID.ValueStringPointer()
	if !plan.Description.IsNull() {
		createProjectRequest.Description = *unleash.NewNullableString(plan.Description.ValueStringPointer())
	}

	project, api_response, err := r.client.ProjectsAPI.CreateProject(context.Background()).CreateProjectSchema(createProjectRequest).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read project ",
			err.Error(),
		)
		return
	}

	if api_response.StatusCode != 201 {
		resp.Diagnostics.AddError(
			"Unexpected HTTP error code received",
			api_response.Status,
		)
		return
	}

	plan.ID = types.StringValue(project.Id)
	plan.Name = types.StringValue(project.Name)
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

	projects, api_response, err := r.client.ProjectsAPI.GetProjects(context.Background()).Execute()

	var project unleash.ProjectSchema
	for _, p := range projects.Projects {
		if p.Id == state.ID.ValueString() {
			project = p
		}
	}

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read Project %s", state.ID.ValueString()),
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

	state.ID = types.StringValue(fmt.Sprintf("%v", project.Id))
	state.Name = types.StringValue(fmt.Sprintf("%v", project.Name))

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
	var state projectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateProjectSchema := *unleash.NewUpdateProjectSchemaWithDefaults()
	updateProjectSchema.Name = *state.Name.ValueStringPointer()
	if !state.Description.IsNull() {
		updateProjectSchema.Description = state.Description.ValueStringPointer()
	}

	req.State.Get(ctx, &state)

	api_response, err := r.client.ProjectsAPI.UpdateProject(context.Background(), state.ID.ValueString()).UpdateProjectSchema(updateProjectSchema).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update project ",
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

	// our update doesn't return the project, so we need to re-read it
	projects, api_response, err := r.client.ProjectsAPI.GetProjects(context.Background()).Execute()

	var project unleash.ProjectSchema
	for _, p := range projects.Projects {
		if p.Id == state.ID.ValueString() {
			project = p
		}
	}
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read Project %s", state.ID.ValueString()),
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

	state.ID = types.StringValue(fmt.Sprintf("%v", project.Id))
	state.Name = types.StringValue(fmt.Sprintf("%v", project.Name))

	if project.Description.IsSet() {
		state.Description = types.StringValue(*project.Description.Get())
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
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

	api_response, err := r.client.ProjectsAPI.DeleteProject(ctx, state.ID.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read project ",
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

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, "Deleted item resource", map[string]any{"success": true})
}
