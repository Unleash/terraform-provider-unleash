package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &projectEnvironmentResource{}
	_ resource.ResourceWithConfigure   = &projectEnvironmentResource{}
	_ resource.ResourceWithImportState = &projectEnvironmentResource{}
)

func NewProjectEnvironmentResource() resource.Resource {
	return &projectEnvironmentResource{}
}

type projectEnvironmentResource struct {
	client *unleash.APIClient
}

type projectEnvironmentResourceModel struct {
	ProjectId             types.String `tfsdk:"project_id"`
	EnvironmentName       types.String `tfsdk:"environment_name"`
	ChangeRequestsEnabled types.Bool   `tfsdk:"change_requests_enabled"`
	RequiredApprovals     types.Int64  `tfsdk:"required_approvals"`
}

func (r *projectEnvironmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		return
	}
	r.client = client
}

type requiredApprovalsValidator struct{}

func (v requiredApprovalsValidator) Description(_ context.Context) string {
	return "Ensures required_approvals is between 1 and 10"
}

func (v requiredApprovalsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v requiredApprovalsValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := req.ConfigValue.ValueInt64()

	if val < 1 || val > 10 {
		resp.Diagnostics.AddError(
			"Invalid required_approvals value",
			fmt.Sprintf("The required_approvals attribute must be between 1 and 10, but got: %d", val),
		)
	}
}

func (r *projectEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment"
}

func (r *projectEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "ProjectEnvironment schema",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "Project identifier.",
				Required:    true,
			},
			"environment_name": schema.StringAttribute{
				Description: "Environment identifier, equivalent to the environment name.",
				Required:    true,
			},
			"change_requests_enabled": schema.BoolAttribute{
				Description: "If change requests are required for this environment, the environment must be enabled for this to have effect.",
				Optional:    true,
				Computed:    true,
			},
			"required_approvals": schema.Int64Attribute{
				Description: "Number of approvals required for change requests.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					requiredApprovalsValidator{},
				},
			},
		},
	}
}

func (r *projectEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import project environment resource")

	// The unique identifier for a project environment is: "<project_id>:<environment_name>"
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected format '<project_id>:<environment_name>'. Example: 'default:development'",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), parts[1])...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Finished importing project environment resource", map[string]any{"success": true})
}

func (r *projectEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Setting project environment config")

	var plan projectEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !r.configureProjectEnvironment(ctx, plan, &resp.Diagnostics) {
		tflog.Warn(ctx, "Failed to configure project environment")
		return
	}

	if !r.hydrateState(ctx, &plan, &resp.Diagnostics) {
		tflog.Warn(ctx, fmt.Sprintf("Failed to hydrate project environment state for project %s", plan.ProjectId.ValueString()))
		return
	}

	plan.normalizeUnmanagedChangeRequestConfig()
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Finished setting project environment", map[string]interface{}{"success": true})
}

func (r *projectEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read project environment change request")

	var state projectEnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectId := state.ProjectId.ValueString()
	envName := state.EnvironmentName.ValueString()

	environments, getEnvironmentsResponse, getEnvironmentsErr := r.client.EnvironmentsAPI.GetProjectEnvironments(ctx, projectId).Execute()

	if !ValidateReadApiResponse(ctx, getEnvironmentsResponse, getEnvironmentsErr, resp, projectId, "Project") {
		return
	}

	enabled := false
	for i := range environments.Environments {
		if environments.Environments[i].Name == envName {
			enabled = true
			break
		}
	}

	if !enabled {
		tflog.Warn(ctx, fmt.Sprintf("Environment %s not found in project %s, removing from state", envName, projectId))
		resp.State.RemoveResource(ctx)
		return
	}

	if !shouldManageChangeRequests(state.ChangeRequestsEnabled, state.RequiredApprovals) {
		state.normalizeUnmanagedChangeRequestConfig()
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		return
	}

	config, getResponse, getErr := r.client.ChangeRequestsAPI.GetProjectChangeRequestConfig(ctx, projectId).Execute()
	if isNotFoundResponse(getResponse) {
		tflog.Debug(ctx, "Change request configuration endpoint is not available for this project environment")
		state.syncChangeRequestConfigNotFound()
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		return
	}

	if !ValidateApiResponse(getResponse, 200, &resp.Diagnostics, getErr) {
		return
	}

	state.syncChangeRequestConfigFromApi(config)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Finished reading project environment change request", map[string]interface{}{"success": true})
}

func (r *projectEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update project environment change request")

	var plan projectEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !r.configureProjectEnvironment(ctx, plan, &resp.Diagnostics) {
		return
	}

	if !r.hydrateState(ctx, &plan, &resp.Diagnostics) {
		return
	}

	plan.normalizeUnmanagedChangeRequestConfig()
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Finished updating project environment change request", map[string]interface{}{"success": true})
}

func (r *projectEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete project environment change request, this will unlink change requests from the relevant project")

	var state projectEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if shouldManageChangeRequests(state.ChangeRequestsEnabled, state.RequiredApprovals) {
		disableChangeRequest := *unleash.NewUpdateChangeRequestEnvironmentConfigSchemaWithDefaults()
		disableChangeRequest.ChangeRequestsEnabled = false
		disableChangeRequest.SetRequiredApprovals(0)

		updateResponse, updateErr := r.client.ChangeRequestsAPI.UpdateProjectChangeRequestConfig(ctx, state.ProjectId.ValueString(), state.EnvironmentName.ValueString()).UpdateChangeRequestEnvironmentConfigSchema(disableChangeRequest).Execute()

		if !isNotFoundResponse(updateResponse) && !ValidateApiResponse(updateResponse, 204, &resp.Diagnostics, updateErr) {
			return
		}
	}

	deleteResponse, err := r.client.ProjectsAPI.RemoveEnvironmentFromProject(ctx, state.ProjectId.ValueString(), state.EnvironmentName.ValueString()).Execute()

	if !ValidateApiResponse(deleteResponse, 200, &resp.Diagnostics, err) {
		return
	}

	resp.State.RemoveResource(ctx)

	tflog.Debug(ctx, "Finished deleting project environment change request", map[string]interface{}{"success": true})
}

func (r *projectEnvironmentResource) configureProjectEnvironment(ctx context.Context, plan projectEnvironmentResourceModel, diagnostics *diag.Diagnostics) bool {
	tflog.Debug(ctx, fmt.Sprintf("Configuring project environment %s for project %s with change requests enabled %t", plan.EnvironmentName.ValueString(), plan.ProjectId.ValueString(), plan.ChangeRequestsEnabled.ValueBool()))
	enabledEnvironmentRequest := *unleash.NewProjectEnvironmentSchemaWithDefaults()
	enabledEnvironmentRequest.Environment = plan.EnvironmentName.ValueString()

	httpResponse, err := r.client.ProjectsAPI.AddEnvironmentToProject(ctx, plan.ProjectId.ValueString()).
		ProjectEnvironmentSchema(enabledEnvironmentRequest).
		Execute()

	if !IsValidApiResponse(httpResponse, []int{200, 409}, diagnostics, err) {
		return false
	}

	if !shouldManageChangeRequests(plan.ChangeRequestsEnabled, plan.RequiredApprovals) {
		return true
	}

	enableChangeRequest := *unleash.NewUpdateChangeRequestEnvironmentConfigSchemaWithDefaults()
	enableChangeRequest.SetChangeRequestsEnabled(plan.ChangeRequestsEnabled.ValueBool())
	if !plan.RequiredApprovals.IsNull() && plan.RequiredApprovals.ValueInt64Pointer() != nil {
		enableChangeRequest.SetRequiredApprovals(int32(*plan.RequiredApprovals.ValueInt64Pointer()))
	}

	updateResponse, updateErr := r.client.ChangeRequestsAPI.UpdateProjectChangeRequestConfig(ctx, plan.ProjectId.ValueString(), plan.EnvironmentName.ValueString()).
		UpdateChangeRequestEnvironmentConfigSchema(enableChangeRequest).
		Execute()

	if !IsValidApiResponse(updateResponse, []int{204, 409}, diagnostics, updateErr) {
		return false
	}
	tflog.Debug(ctx, fmt.Sprintf("Successfully configured project environment %s for project %s with change requests enabled %t", plan.EnvironmentName.ValueString(), plan.ProjectId.ValueString(), plan.ChangeRequestsEnabled.ValueBool()))
	return true
}

func (r *projectEnvironmentResource) hydrateState(ctx context.Context, state *projectEnvironmentResourceModel, diagnostics *diag.Diagnostics) bool {
	enabled, ok := r.projectEnvironmentIsEnabled(ctx, state.ProjectId.ValueString(), state.EnvironmentName.ValueString(), diagnostics)
	if !ok {
		return false
	}

	if !enabled {
		diagnostics.AddError(
			"Environment not found in project",
			fmt.Sprintf("Environment %s is not enabled for project %s", state.EnvironmentName.ValueString(), state.ProjectId.ValueString()),
		)
		return false
	}

	if !shouldManageChangeRequests(state.ChangeRequestsEnabled, state.RequiredApprovals) {
		state.normalizeUnmanagedChangeRequestConfig()
		return true
	}

	return r.hydrateManagedChangeRequestState(ctx, state, diagnostics)
}

func (r *projectEnvironmentResource) projectEnvironmentIsEnabled(ctx context.Context, projectId string, envName string, diagnostics *diag.Diagnostics) (bool, bool) {
	environments, getEnvironmentsResponse, getEnvironmentsErr := r.client.EnvironmentsAPI.GetProjectEnvironments(ctx, projectId).Execute()
	if !ValidateApiResponse(getEnvironmentsResponse, 200, diagnostics, getEnvironmentsErr) {
		return false, false
	}

	for _, environment := range environments.Environments {
		if environment.Name == envName {
			return true, true
		}
	}

	return false, true
}

func (r *projectEnvironmentResource) hydrateManagedChangeRequestState(ctx context.Context, state *projectEnvironmentResourceModel, diagnostics *diag.Diagnostics) bool {
	config, getResponse, getErr := r.client.ChangeRequestsAPI.GetProjectChangeRequestConfig(ctx, state.ProjectId.ValueString()).Execute()
	if isNotFoundResponse(getResponse) {
		state.syncChangeRequestConfigNotFound()
		return true
	}

	if !ValidateApiResponse(getResponse, 200, diagnostics, getErr) {
		return false
	}

	state.syncChangeRequestConfigFromApi(config)
	return true
}

func (m *projectEnvironmentResourceModel) hydrateResponseFromApi(config []unleash.ChangeRequestEnvironmentConfigSchema) {
	var envChangeRequestConfig *unleash.ChangeRequestEnvironmentConfigSchema

	for _, env := range config {
		if env.Environment == m.EnvironmentName.ValueString() {
			envChangeRequestConfig = &env
			break
		}
	}

	if envChangeRequestConfig == nil {
		m.ChangeRequestsEnabled = types.BoolValue(false)
		m.RequiredApprovals = types.Int64Null()
		return
	}

	var requiredApprovals basetypes.Int64Value

	if envChangeRequestConfig.RequiredApprovals.IsSet() && envChangeRequestConfig.RequiredApprovals.Get() != nil {
		requiredApprovals = types.Int64Value(int64(*envChangeRequestConfig.RequiredApprovals.Get()))
	} else {
		requiredApprovals = types.Int64Null()
	}

	m.ProjectId = types.StringValue(m.ProjectId.ValueString())
	m.EnvironmentName = types.StringValue(m.EnvironmentName.ValueString())
	m.ChangeRequestsEnabled = types.BoolValue(envChangeRequestConfig.ChangeRequestEnabled)
	m.RequiredApprovals = requiredApprovals
}

func (m *projectEnvironmentResourceModel) syncChangeRequestConfigFromApi(config []unleash.ChangeRequestEnvironmentConfigSchema) {
	if !shouldManageChangeRequests(m.ChangeRequestsEnabled, m.RequiredApprovals) {
		return
	}

	m.hydrateResponseFromApi(config)
}

func (m *projectEnvironmentResourceModel) syncChangeRequestConfigNotFound() {
	if !shouldManageChangeRequests(m.ChangeRequestsEnabled, m.RequiredApprovals) {
		return
	}

	m.resetChangeRequestConfig()
}

func (m *projectEnvironmentResourceModel) resetChangeRequestConfig() {
	m.ChangeRequestsEnabled = types.BoolValue(false)
	m.RequiredApprovals = types.Int64Null()
}

func (m *projectEnvironmentResourceModel) normalizeUnmanagedChangeRequestConfig() {
	if shouldManageChangeRequests(m.ChangeRequestsEnabled, m.RequiredApprovals) {
		return
	}

	if m.ChangeRequestsEnabled.IsUnknown() {
		m.ChangeRequestsEnabled = types.BoolNull()
	}
	if m.RequiredApprovals.IsUnknown() {
		m.RequiredApprovals = types.Int64Null()
	}
}

func shouldManageChangeRequests(changeRequestsEnabled types.Bool, requiredApprovals types.Int64) bool {
	return (!changeRequestsEnabled.IsNull() && !changeRequestsEnabled.IsUnknown()) || (!requiredApprovals.IsNull() && !requiredApprovals.IsUnknown())
}

func isNotFoundResponse(response *http.Response) bool {
	return response != nil && response.StatusCode == http.StatusNotFound
}
