package provider

import (
	"context"
	"fmt"
	"strconv"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	Id        types.String `tfsdk:"id"`
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
				Description: "Send a welcome email to the customer or not. Defaults to false",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import user resource")

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Debug(ctx, "Finished importing user data source", map[string]any{"success": true})
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create user resource")
	var plan userResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	roleId, diags := plan.RootRole.ToInt64Value(ctx)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	roleId32 := int32(roleId.ValueInt64())

	// Generate API request body from plan
	createUserRequest := *unleash.NewCreateUserSchemaWithDefaults()
	createUserRequest.Name = plan.Name.ValueStringPointer()
	createUserRequest.Username = plan.Username.ValueStringPointer()
	createUserRequest.Email = plan.Email.ValueStringPointer()
	createUserRequest.RootRole = unleash.Int32AsCreateUserSchemaRootRole(&roleId32)
	createUserRequest.Password = plan.Password.ValueStringPointer()
	// Should SendEmail be part of the state? How do we model ephimeral input state in terraform?
	createUserRequest.SendEmail = plan.SendEmail.ValueBoolPointer()
	// do we need to expose the invite link if send email is false?

	user, api_response, err := r.client.UsersAPI.CreateUser(ctx).CreateUserSchema(createUserRequest).Execute()

	if !ValidateApiResponse(api_response, 201, &resp.Diagnostics, err) {
		return
	}

	// Update model with response
	plan.Id = types.StringValue(strconv.Itoa(int(user.Id)))

	plan.RootRole = types.Int64Value(int64(*user.RootRole.Int32))
	if user.Username.IsSet() {
		plan.Username = types.StringValue(*user.Username.Get())
	} else {
		plan.Username = types.StringNull()
	}
	if user.Email != nil {
		plan.Email = types.StringValue(*user.Email)
	} else {
		plan.Email = types.StringNull()
	}
	if user.Name.IsSet() {
		plan.Name = types.StringValue(*user.Name.Get())
	} else {
		plan.Name = types.StringNull()
	}
	// TODO note the output state is not the same as input state
	// here in output state we're saying what happened (i.e. Id is present)
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

	userId, err := strconv.Atoi(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("User id was not a number %s", state.Id.ValueString()),
			err.Error(),
		)
		return
	}

	// Get fresh data
	user, httpResponse, err := r.client.UsersAPI.GetUser(ctx, int32(userId)).Execute()

	if !ValidateReadApiResponse(ctx, httpResponse, err, resp, state.Id.ValueString(), "User") {
		return
	}

	state.Id = types.StringValue(strconv.Itoa(int(user.Id)))

	if user.Email != nil {
		state.Email = types.StringValue(*user.Email)
	} else {
		state.Email = types.StringNull()
	}
	if user.Username.IsSet() {
		state.Username = types.StringValue(*user.Username.Get())
	} else {
		state.Username = types.StringNull()
	}
	if user.Name.IsSet() {
		state.Name = types.StringValue(*user.Name.Get())
	} else {
		state.Name = types.StringNull()
	}
	if state.SendEmail.IsNull() || state.SendEmail.IsUnknown() {
		state.SendEmail = types.BoolValue(false)
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
	// this is the error we generate right now:
	// │ Error: Provider produced inconsistent result after apply
	// │
	// │ When applying changes to unleash_user.chuck, provider "provider[\"registry.terraform.io/unleash/unleash\"]" produced an unexpected new value: .username: was null, but now cty.StringVal("chuck").
	// │
	// │ This is a bug in the provider, which should be reported in the provider's own issue tracker.

	if resp.Diagnostics.HasError() {
		return
	}

	newRootRole := int32(state.RootRole.ValueInt64())
	role := unleash.Int32AsCreateUserSchemaRootRole(&newRootRole)

	updateUserSchema := *unleash.NewUpdateUserSchemaWithDefaults()
	updateUserSchema.Name = state.Name.ValueStringPointer()
	updateUserSchema.Email = state.Email.ValueStringPointer()
	updateUserSchema.RootRole = &role

	req.State.Get(ctx, &state) // the id is part of the state, not the plan, this is how we get its value

	id, err := strconv.Atoi(state.Id.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("User id was not a number %s", state.Id.ValueString()),
			err.Error(),
		)
		return
	}

	user, api_response, err := r.client.UsersAPI.UpdateUser(ctx, int32(id)).UpdateUserSchema(updateUserSchema).Execute()

	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	// Update model with response
	if user.Email != nil {
		state.Email = types.StringValue(*user.Email)
	} else {
		state.Email = types.StringNull()
	}
	if user.Username.IsSet() {
		state.Username = types.StringValue(*user.Username.Get())
	} else {
		state.Username = types.StringNull()
	}
	if user.Name.IsSet() {
		state.Name = types.StringValue(*user.Name.Get())
	} else {
		state.Name = types.StringNull()
	}
	state.RootRole = types.Int64Value(int64(*user.RootRole.Int32))

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

	userId, err := strconv.Atoi(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("User id was not a number %s", state.Id.ValueString()),
			err.Error(),
		)
		return
	}

	api_response, err := r.client.UsersAPI.DeleteUser(ctx, int32(userId)).Execute()

	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, "Deleted item resource", map[string]any{"success": true})
}
