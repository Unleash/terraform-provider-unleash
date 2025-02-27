package provider

import (
	"context"

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
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
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
		Description: "Fetch a context field.",
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

	environment, apiResponse, err := r.client.EnvironmentsAPI.CreateEnvironment(ctx).CreateEnvironmentSchema(createEnvironmentRequest).Execute()

	if !ValidateApiResponse(apiResponse, 201, &resp.Diagnostics, err) {
		return
	}

	plan = environmentResourceModel{
		Name: types.StringValue(environment.Name),
		Type: types.StringValue(environment.Type),
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

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	state = environmentResourceModel{
		Name: types.StringValue(environment.Name),
		Type: types.StringValue(environment.Type),
	}

	resp.State.Set(ctx, &state)

	tflog.Debug(ctx, "Finished reading environment resource", map[string]interface{}{"success": true})
}

func (r *environmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update environment resource")
	var plan environmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateEnvironmentRequest := *unleash.NewUpdateEnvironmentSchemaWithDefaults()
	updateEnvironmentRequest.SetType(plan.Type.ValueString())

	environment, apiResponse, err := r.client.EnvironmentsAPI.UpdateEnvironment(ctx, plan.Name.ValueString()).UpdateEnvironmentSchema(updateEnvironmentRequest).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	plan = environmentResourceModel{
		Name: types.StringValue(environment.Name),
		Type: types.StringValue(environment.Type),
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
