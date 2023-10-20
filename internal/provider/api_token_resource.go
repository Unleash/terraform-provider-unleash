package provider

import (
	"context"
	"fmt"
	"time"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &apiTokenResource{}
	_ resource.ResourceWithConfigure = &apiTokenResource{}
)

// NewApiTokenResource is a helper function to simplify the provider implementation.
func NewApiTokenResource() resource.Resource {
	return &apiTokenResource{}
}

// apiTokenResource is the data source implementation.
type apiTokenResource struct {
	client *unleash.APIClient
}

type apiTokenResourceModel struct {
	Secret types.String `tfsdk:"secret"`
	// A unique name for this particular token
	TokenName types.String `tfsdk:"token_name"`
	// The type of API token
	Type types.String `tfsdk:"type"`
	// The environment the token has access to. `*` if it has access to all environments.
	Environment types.String `tfsdk:"environment"`
	// The project this token belongs to.
	Project types.String `tfsdk:"project"`
	// The list of projects this token has access to. If the token has access to specific projects they will be listed here. If the token has access to all projects it will be represented as `[*]`
	Projects types.List `tfsdk:"projects"`
	// The token's expiration date. NULL if the token doesn't have an expiration set.
	ExpiresAt types.String `tfsdk:"expires_at"`
}

// Configure adds the provider configured client to the data source.
func (r *apiTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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
func (r *apiTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_token"
}

// Schema defines the schema for the data source. TODO: can we transform OpenAPI schema into TF schema?
func (r *apiTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "ApiToken schema",
		Attributes: map[string]schema.Attribute{
			"secret": schema.StringAttribute{
				Description: "Secret token value.",
				Computed:    true,
				Sensitive:   true,
			},
			"token_name": schema.StringAttribute{
				Description: "The name of the token.",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the token.",
				Optional:    true,
			},
			"environment": schema.StringAttribute{
				Description: "An environment the token has access to.",
				Optional:    true,
				Computed:    true,
			},
			"project": schema.StringAttribute{
				Description: "A project the token belongs to.",
				Optional:    true,
				Computed:    true,
			},
			"projects": schema.ListAttribute{
				Description: "The list of projects this token has access to. If the token has access to specific projects they will be listed here. If the token has access to all projects it will be represented as `[*]`.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"expires_at": schema.StringAttribute{
				Description: "When the token expires",
				Optional:    true,
			},
		},
	}
}

func (r *apiTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create api token resource")
	var plan apiTokenResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createAPITokenRequest := *unleash.NewCreateApiTokenSchemaOneOf2(plan.Type.ValueString(), plan.TokenName.ValueString())
	if !plan.Environment.IsNull() && !plan.Environment.IsUnknown() {
		createAPITokenRequest.Environment = plan.Environment.ValueStringPointer()
	}
	project := plan.Project.ValueString()
	if project != "" {
		createAPITokenRequest.Project = &project
	}
	if !plan.Projects.IsNull() && !plan.Projects.IsUnknown() {
		tflog.Debug(ctx, fmt.Sprintf("Iterating over projects: %+v to put them into %+v", plan.Projects, createAPITokenRequest.Projects))
		plan.Projects.ElementsAs(ctx, &createAPITokenRequest.Projects, false)
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		expire, err := time.Parse(time.RFC3339, plan.ExpiresAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to parse expiration date",
				err.Error(),
			)
			return
		}
		createAPITokenRequest.ExpiresAt = &expire
	}
	createTokenRequest := unleash.CreateApiTokenSchemaOneOf2AsCreateApiTokenSchema(&createAPITokenRequest)

	token, api_response, err := r.client.APITokensAPI.CreateApiToken(ctx).CreateApiTokenSchema(createTokenRequest).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Api Token",
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

	// Update model with response
	tflog.Debug(ctx, fmt.Sprintf("Created token: %+v", token))
	var newState apiTokenResourceModel
	newState.Secret = types.StringValue(token.Secret)
	newState.TokenName = types.StringValue(token.TokenName)
	newState.Type = types.StringValue(token.Type)
	if token.Environment != nil {
		newState.Environment = types.StringValue(*token.Environment)
	} else {
		newState.Environment = types.StringNull()
	}
	if token.ExpiresAt.IsSet() && token.ExpiresAt.Get() != nil {
		newState.ExpiresAt = types.StringValue(token.ExpiresAt.Get().Format(time.RFC3339))
	} else {
		newState.ExpiresAt = types.StringNull()
	}
	newState.Project = types.StringValue(token.Project)
	if token.Projects == nil {
		// replace with response
		tflog.Debug(ctx, fmt.Sprintf("Token has projects: %+v but plan is %+v", token.Projects, plan.Projects))
	} else {
		if token.Projects != nil {
			newState.Projects, _ = basetypes.NewListValueFrom(ctx, types.StringType, token.Projects)
		}
		tflog.Debug(ctx, fmt.Sprintf("Projects not null: %+v", token.Projects))
	}

	// Set state
	if resp.Diagnostics.HasError() {
		fmt.Printf("Before set state: %+v\n", newState)
		tflog.Debug(ctx, fmt.Sprintf("Before set state: %+v", newState))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "After set state")
		return
	}
	tflog.Debug(ctx, "Finished creating api token data source", map[string]any{"success": true})
}

func (r *apiTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read api token resource")
	var state apiTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tokens, api_response, err := r.client.APITokensAPI.GetAllApiTokens(context.Background()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read Api Token %s", state.Secret.ValueString()),
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
	var token unleash.ApiTokenSchema
	for _, t := range tokens.Tokens {
		if t.Secret == state.Secret.ValueString() {
			token = t
		}
	}

	// Update model with response
	state.Secret = types.StringValue(token.Secret)
	state.Environment = types.StringValue(*token.Environment)
	state.TokenName = types.StringValue(token.TokenName)
	state.Type = types.StringValue(token.Type)
	state.Project = types.StringValue(token.Project)
	if token.Projects != nil {
		state.Projects, _ = basetypes.NewListValueFrom(ctx, types.StringType, token.Projects)
	}
	if token.ExpiresAt.IsSet() && token.ExpiresAt.Get() != nil {
		state.ExpiresAt = types.StringValue(token.ExpiresAt.Get().Format(time.RFC3339))
	} else {
		state.ExpiresAt = types.StringNull()
	}
	tflog.Debug(ctx, "Finished populating model", map[string]any{"success": true})

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading api token data source", map[string]any{"success": true})
}

func (r *apiTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update api token resource")
	var state apiTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	var expire time.Time
	var err error
	if !state.ExpiresAt.IsNull() && !state.ExpiresAt.IsUnknown() {
		expire, err = time.Parse(time.RFC3339, state.ExpiresAt.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to parse expiration date",
				err.Error(),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError("ExpiresAt is mandatory when updating a token", "The value provided was null or unknown")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	updateApiTokenSchema := *unleash.NewUpdateApiTokenSchema(expire)

	req.State.Get(ctx, &state) // the id is part of the state, not the plan, this is how we get its value

	api_response, err := r.client.APITokensAPI.UpdateApiToken(context.Background(), state.Secret.ValueString()).UpdateApiTokenSchema(updateApiTokenSchema).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Api Token",
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

	// Set state
	state.ExpiresAt = types.StringValue(expire.Format(time.RFC3339))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished updating api token data source", map[string]any{"success": true})
}

func (r *apiTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete api token")
	var state apiTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	api_response, err := r.client.APITokensAPI.DeleteApiToken(ctx, state.Secret.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete User",
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
