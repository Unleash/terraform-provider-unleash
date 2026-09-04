package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &environmentResource{}
	_ resource.ResourceWithConfigure   = &environmentResource{}
	_ resource.ResourceWithImportState = &environmentResource{}
)

func NewEnvironmentResource() resource.Resource {
	return &environmentResource{}
}

type environmentResource struct {
	client *unleash.APIClient
}

type environmentResourceModel struct {
	Name              types.String `tfsdk:"name"`
	Type              types.String `tfsdk:"type"`
	RequiredApprovals types.Int64  `tfsdk:"required_approvals"`
}

func (r *environmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		return
	}
	r.client = client
}

func (r *environmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *environmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Unleash environments.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the environment. Must be a URL-friendly string according to RFC 3968. " +
					"Changing this property will require the resource to be replaced, it's generally safer to remove this resource and create a new one.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of the environment. Unleash recognizes 'development', 'test', 'preproduction' and 'production'. " +
					"You can pass other values and Unleash will accept them but they will carry no special semantics.",
				Required: true,
			},
			"required_approvals": schema.Int64Attribute{
				Description: "The number of approvals a change request must collect before it can be applied in this environment. " +
					"Setting it turns on environment-level change requests, and every project that uses this environment inherits the value. " +
					"Projects can still override it unless they have no members allowed to update the project. " +
					"Leave it unset to not preconfigure change requests for this environment. " +
					"Requires Unleash Enterprise 6.10 or later.",
				Optional: true,
				Validators: []validator.Int64{
					requiredApprovalsValidator{},
				},
			},
		},
	}
}

func (r *environmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import environment resource")

	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)

	tflog.Debug(ctx, "Finished importing environment data source", map[string]interface{}{"success": true})
}

func (r *environmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create environment resource")
	var plan environmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createEnvironmentRequest := *unleash.NewCreateEnvironmentSchemaWithDefaults()
	createEnvironmentRequest.Name = plan.Name.ValueString()
	createEnvironmentRequest.Type = plan.Type.ValueString()
	if hasRequiredApprovals(plan.RequiredApprovals) {
		createEnvironmentRequest.SetRequiredApprovals(int32(plan.RequiredApprovals.ValueInt64()))
	}

	environment, apiResponse, err := r.client.EnvironmentsAPI.CreateEnvironment(ctx).CreateEnvironmentSchema(createEnvironmentRequest).Execute()

	if !ValidateApiResponse(apiResponse, 201, &resp.Diagnostics, err) {
		return
	}

	requestedApprovals := plan.RequiredApprovals
	plan.hydrateFromApi(*environment)
	if !requiredApprovalsWereApplied(requestedApprovals, plan.RequiredApprovals, &resp.Diagnostics) {
		return
	}

	resp.State.Set(ctx, &plan)
	tflog.Debug(ctx, "Finished creating environment resource", map[string]interface{}{"success": true})

}

func (r *environmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read environment resource")
	var state environmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	environment, apiResponse, err := r.client.EnvironmentsAPI.GetEnvironment(ctx, state.Name.ValueString()).Execute()

	if !ValidateReadApiResponse(ctx, apiResponse, err, resp, state.Name.ValueString(), "Environment") {
		return
	}

	state.hydrateFromApi(*environment)

	resp.State.Set(ctx, &state)

	tflog.Debug(ctx, "Finished reading environment resource", map[string]interface{}{"success": true})
}

func (r *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update environment resource")
	var plan environmentResourceModel
	var state environmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateEnvironmentRequest := *unleash.NewUpdateEnvironmentSchemaWithDefaults()
	updateEnvironmentRequest.SetType(plan.Type.ValueString())
	applyRequiredApprovalsUpdate(&updateEnvironmentRequest, plan.RequiredApprovals, state.RequiredApprovals)

	environment, apiResponse, err := r.client.EnvironmentsAPI.UpdateEnvironment(ctx, plan.Name.ValueString()).UpdateEnvironmentSchema(updateEnvironmentRequest).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	requestedApprovals := plan.RequiredApprovals
	plan.hydrateFromApi(*environment)
	if !requiredApprovalsWereApplied(requestedApprovals, plan.RequiredApprovals, &resp.Diagnostics) {
		return
	}

	resp.State.Set(ctx, &plan)
	tflog.Debug(ctx, "Finished updating environment resource", map[string]interface{}{"success": true})
}

func (r *environmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete environment resource")
	var state environmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResponse, err := r.client.EnvironmentsAPI.RemoveEnvironment(ctx, state.Name.ValueString()).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, "Finished deleting environment resource", map[string]interface{}{"success": true})
}

func (m *environmentResourceModel) hydrateFromApi(api unleash.EnvironmentSchema) {
	m.Name = types.StringValue(api.Name)
	m.Type = types.StringValue(api.Type)
	m.RequiredApprovals = requiredApprovalsFromApi(api.RequiredApprovals)
}

// Sends requiredApprovals only when Terraform manages it: instances without environment-level
// change requests reject the field outright, and clearing it needs an explicit null. An unknown
// planned value also clears, which is unreachable while the attribute is Optional and not
// Computed; adding Computed would make it reachable and would need handling here.
func applyRequiredApprovalsUpdate(request *unleash.UpdateEnvironmentSchema, planned types.Int64, current types.Int64) {
	if hasRequiredApprovals(planned) {
		request.SetRequiredApprovals(int32(planned.ValueInt64()))
		return
	}

	if hasRequiredApprovals(current) {
		request.SetRequiredApprovalsNil()
	}
}

func requiredApprovalsFromApi(requiredApprovals unleash.NullableInt32) types.Int64 {
	if !requiredApprovals.IsSet() || requiredApprovals.Get() == nil {
		return types.Int64Null()
	}

	// Unleash treats zero approvals as "not preconfigured" and the schema's minimum of 1 means a
	// configuration can never express 0, so keep state on a value it can converge on.
	if *requiredApprovals.Get() == 0 {
		return types.Int64Null()
	}

	return types.Int64Value(int64(*requiredApprovals.Get()))
}

// The attribute is Optional and not Computed, so Terraform requires state to match the
// configuration after apply. Diagnosing a silently ignored field here replaces the framework's
// opaque "provider produced inconsistent result after apply".
func requiredApprovalsWereApplied(requested types.Int64, applied types.Int64, diagnostics *diag.Diagnostics) bool {
	if requested.Equal(applied) {
		return true
	}

	diagnostics.AddError(
		"Unleash did not apply required_approvals",
		fmt.Sprintf(
			"Requested %s but the environment reports %s. Environment-level change requests require "+
				"Unleash Enterprise 6.10 or later; remove required_approvals if this instance does not support it.",
			requested, applied,
		),
	)

	return false
}

func hasRequiredApprovals(requiredApprovals types.Int64) bool {
	return !requiredApprovals.IsNull() && !requiredApprovals.IsUnknown()
}
