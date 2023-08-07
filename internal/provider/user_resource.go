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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

// NewUserResource is a helper function to simplify the provider implementation.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// userResource is the data source implementation.
type userResource struct {
	client *unleash.APIClient
}

type userResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Username  types.String `tfsdk:"username"`
	Email     types.String `tfsdk:"email"`
	Name      types.String `tfsdk:"name"`
	Password  types.String `tfsdk:"password"`
	RootRole  types.Int64  `tfsdk:"root_role"`
	SendEmail types.Bool   `tfsdk:"send_email"`
}

// Configure adds the provider configured client to the data source.
func (r *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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
func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the data source. TODO: can we transform OpenAPI schema into TF schema?
func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "User schema",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this user.",
				Computed:    true,
			},
			// TODO define optional either username or email, not both nil
			"username": schema.StringAttribute{
				Description: "The username.",
				Optional:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email of the user.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the user.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The password of the user.",
				Optional:    true,
				Sensitive:   true,
			},
			"root_role": schema.Int64Attribute{
				Description: "The role id for the user.",
				Required:    true,
			},
			"send_email": schema.BoolAttribute{
				Description: "Send a welcome email to the customer or not. Defaults to true",
				Optional:    true,
			},
		},
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import user resource")
	var state userResourceModel

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished importing user data source", map[string]any{"success": true})
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create user resource")
	var plan userResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createUserRequest := *unleash.NewCreateUserSchemaWithDefaults()
	createUserRequest.Name = plan.Name.ValueStringPointer()
	createUserRequest.Username = plan.Username.ValueStringPointer()
	createUserRequest.Email = plan.Email.ValueStringPointer()
	createUserRequest.RootRole = float32(plan.RootRole.ValueInt64())
	// Should SendEmail be part of the state? How do we model ephimeral input state in terraform?
	createUserRequest.SendEmail = plan.SendEmail.ValueBoolPointer()
	// do we need to expose the invite link if send email is false?

	user, api_response, err := r.client.UsersApi.CreateUser(context.Background()).CreateUserSchema(createUserRequest).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read User",
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
	plan.ID = types.StringValue(fmt.Sprintf("%v", user.Id))
	plan.RootRole = types.Int64Value(int64(*user.RootRole))
	if user.Username != nil {
		plan.Username = types.StringValue(*user.Username)
	} else {
		plan.Username = types.StringNull()
	}
	if user.Email != nil {
		plan.Email = types.StringValue(*user.Email)
	} else {
		plan.Email = types.StringNull()
	}
	if user.Name != nil {
		plan.Name = types.StringValue(*user.Name)
	} else {
		plan.Name = types.StringNull()
	}
	// TODO note the output state is not the same as input state
	// here in output state we're saying what happened (i.e. ID is present)
	// but in the input state we don't know if the email was sent or not
	// but we do have a SendEmail configuration
	// In the output we receive if we've sent the email or not

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished creating user data source", map[string]any{"success": true})
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read user resource")
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get fresh data
	user, api_response, err := r.client.UsersApi.GetUser(context.Background(), state.ID.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read User %s", state.ID.ValueString()),
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

	// Update model with response
	state.ID = types.StringValue(fmt.Sprintf("%v", user.Id))
	if user.Email != nil {
		state.Email = types.StringValue(*user.Email)
	} else {
		state.Email = types.StringNull()
	}
	if user.Username != nil {
		state.Username = types.StringValue(*user.Username)
	} else {
		state.Username = types.StringNull()
	}
	if user.Name != nil {
		state.Name = types.StringValue(*user.Name)
	} else {
		state.Name = types.StringNull()
	}
	state.RootRole = types.Int64Value(int64(*user.RootRole))
	tflog.Debug(ctx, "Finished populating model", map[string]any{"success": true})

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading user data source", map[string]any{"success": true})
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update user resource")
	var state userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	// TODO fail if you try to change the username, that's not possible or let the server fail?

	if resp.Diagnostics.HasError() {
		return
	}

	newRootRole := float32(state.RootRole.ValueInt64())
	updateUserSchema := *unleash.NewUpdateUserSchemaWithDefaults()
	updateUserSchema.Name = state.Name.ValueStringPointer()
	updateUserSchema.Email = state.Email.ValueStringPointer()
	updateUserSchema.RootRole = &newRootRole
	requestBody, err := updateUserSchema.ToMap()
	// handle if error
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read User",
			err.Error(),
		)
		return
	}

	req.State.Get(ctx, &state) // I still don't get why but this is needed and the req.Plan.Get above is also needed and the order has to be this one... Otherwise state.ID seems to be null

	user, api_response, err := r.client.UsersApi.UpdateUser(context.Background(), state.ID.ValueString()).RequestBody(requestBody).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read User",
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

	// Update model with response
	if user.Email != nil {
		state.Email = types.StringValue(*user.Email)
	} else {
		state.Email = types.StringNull()
	}
	if user.Username != nil {
		state.Username = types.StringValue(*user.Username)
	} else {
		state.Username = types.StringNull()
	}
	if user.Name != nil {
		state.Name = types.StringValue(*user.Name)
	} else {
		state.Name = types.StringNull()
	}
	state.RootRole = types.Int64Value(int64(*user.RootRole))

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished updating user data source", map[string]any{"success": true})
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete user")
	var state userResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	api_response, err := r.client.UsersApi.DeleteUser(ctx, state.ID.ValueString()).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read User",
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
