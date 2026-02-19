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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &groupResource{}
	_ resource.ResourceWithConfigure = &groupResource{}
)

// NewGroupResource is a helper function to simplify the provider implementation.
func NewGroupResource() resource.Resource {
	return &groupResource{}
}

// groupResource is the resource implementation.
type groupResource struct {
	client *unleash.APIClient
}

type groupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	MappingsSSO types.List   `tfsdk:"mappings_sso"`
	RootRole    types.Int64  `tfsdk:"root_role"`
	Users       types.List   `tfsdk:"users"`
}

// Helper function to convert API users to Terraform model.
func convertAPIUsersToModel(apiUsers []unleash.GroupUserModelSchema) []int64 {
	var usersIDs []int64
	for _, apiUser := range apiUsers {
		if apiUser.User.Id != 0 {
			uID := int64(apiUser.User.Id)
			usersIDs = append(usersIDs, uID)
		}
	}
	return usersIDs
}

// Helper function to convert Terraform user models to API format.
func convertModelUsersToAPI(ctx context.Context, usersList types.List, diagnostics *diag.Diagnostics) []unleash.CreateGroupSchemaUsersInner {
	var usersIDs []int64
	diagnostics.Append(usersList.ElementsAs(ctx, &usersIDs, false)...)
	if diagnostics.HasError() {
		return nil
	}
	var usersAPI []unleash.CreateGroupSchemaUsersInner
	for _, userID := range usersIDs {
		userAPI := unleash.CreateGroupSchemaUsersInner{
			User: unleash.CreateGroupSchemaUsersInnerUser{
				Id: int32(userID),
			},
		}
		usersAPI = append(usersAPI, userAPI)
	}
	return usersAPI
}

// Helper function to populate group state from API response.
func populateGroupStateFromAPI(ctx context.Context, group *unleash.GroupSchema, state *groupResourceModel, diagnostics *diag.Diagnostics) {
	if group == nil {
		tflog.Error(ctx, "populateGroupStateFromAPI: group is nil")
		diagnostics.AddError("Nil Group Response", "The API returned a nil group response")
		return
	}
	// ID
	state.ID = types.StringValue(fmt.Sprint(*group.Id))

	// Name
	state.Name = types.StringValue(group.Name)

	// Description
	if group.Description.IsSet() {
		state.Description = types.StringValue(*group.Description.Get())
	} else {
		state.Description = types.StringNull()
	}

	// RootRole.
	if group.RootRole.IsSet() {
		rolePtr := group.RootRole.Get()
		if rolePtr != nil {
			state.RootRole = types.Int64Value(int64(*rolePtr))
		} else {
			state.RootRole = types.Int64Null()
		}
	}

	// MappingsSSO.
	if len(group.MappingsSSO) > 0 {
		mappingSSO, diags := types.ListValueFrom(ctx, types.StringType, group.MappingsSSO)
		diagnostics.Append(diags...)
		state.MappingsSSO = mappingSSO
	} else {
		state.MappingsSSO = types.ListNull(types.StringType)
	}
	tflog.Debug(ctx, "populateGroupStateFromAPI: Processing users", map[string]any{
		"users_nil": group.Users == nil,
		"users_len": len(group.Users),
	})
	// Users.
	if len(group.Users) > 0 {
		usersIDs := convertAPIUsersToModel(group.Users)

		usersList, diags := types.ListValueFrom(ctx, types.Int64Type, usersIDs)
		diagnostics.Append(diags...)
		state.Users = usersList
	} else {
		state.Users = types.ListNull(types.Int64Type)
	}
}

// Configure adds the provider configured client to the resource.
func (r *groupResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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

// Metadata returns the resource type name.
func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Group schema",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this group",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name for this group",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the group's purpose.",
				Optional:    true,
			},
			"mappings_sso": schema.ListAttribute{
				Description: "SSO group mappings for this group.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"root_role": schema.Int64Attribute{
				Description: "The root role ID for this group.",
				Optional:    true,
			},
			"users": schema.ListAttribute{
				Description: "List of user IDs to add to this group",
				Optional:    true,
				ElementType: types.Int64Type,
			},
		},
	}
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import group resource")
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	tflog.Debug(ctx, "Finished importing group resource", map[string]any{"success": true})
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create group resource")
	var plan groupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request
	createGroupRequest := *unleash.NewCreateGroupSchemaWithDefaults()
	// Required: Name
	createGroupRequest.Name = plan.Name.ValueString()

	// Optional: Description
	if !plan.Description.IsNull() {
		createGroupRequest.Description = *unleash.NewNullableString(plan.Description.ValueStringPointer())
	}

	// Optional: RootRole
	if !plan.RootRole.IsNull() {
		roleID := float32(plan.RootRole.ValueInt64())
		createGroupRequest.RootRole = *unleash.NewNullableFloat32(&roleID)
	}

	// Optional: MappingsSSO
	if !plan.MappingsSSO.IsNull() && len(plan.MappingsSSO.Elements()) > 0 {
		var mappings []string
		resp.Diagnostics.Append(plan.MappingsSSO.ElementsAs(ctx, &mappings, false)...)
		if !resp.Diagnostics.HasError() {
			createGroupRequest.MappingsSSO = mappings
		}
	}

	// Optional: Users
	if !plan.Users.IsNull() && len(plan.Users.Elements()) > 0 {
		usersAPI := convertModelUsersToAPI(ctx, plan.Users, &resp.Diagnostics)
		if !resp.Diagnostics.HasError() {
			createGroupRequest.Users = usersAPI
		}
	}

	// Execute API call
	// NOTE: Create does not return the user list as specified in the API spec, we need a Read to obtain the users
	group, apiResponse, err := r.client.UsersAPI.CreateGroup(ctx).CreateGroupSchema(createGroupRequest).Execute()
	if !ValidateApiResponse(apiResponse, 201, &resp.Diagnostics, err) {
		return
	}
	groupID := fmt.Sprint(*group.Id)
	// Read to get the group
	createdGroup, apiReadResponse, err := r.client.UsersAPI.GetGroup(ctx, groupID).Execute()
	// Homemade error checking for Read request in create
	if err != nil || apiReadResponse.StatusCode != 200 {
		return
	}
	// Populate state from API response
	populateGroupStateFromAPI(ctx, createdGroup, &plan, &resp.Diagnostics)

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished creating group resource", map[string]any{"success": true})
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read group resource")

	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the group from the server
	group, httpResp, err := r.client.UsersAPI.GetGroup(ctx, state.ID.ValueString()).Execute()
	if !ValidateReadApiResponse(ctx, httpResp, err, resp, state.ID.ValueString(), "Group") {
		return
	}

	// Populate state from API response
	populateGroupStateFromAPI(ctx, group, &state, &resp.Diagnostics)

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading group resource", map[string]any{"success": true})
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update group resource")
	// State
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Get the ID from the state
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Build API request
	updateGroupRequest := *unleash.NewCreateGroupSchemaWithDefaults() // API uses same schema for update
	updateGroupRequest.Name = plan.Name.ValueString()

	// Optional: Description
	if !plan.Description.IsNull() {
		updateGroupRequest.Description = *unleash.NewNullableString(plan.Description.ValueStringPointer())
	}

	// Optional: RootRole
	if !plan.RootRole.IsNull() {
		roleID := float32(plan.RootRole.ValueInt64())
		updateGroupRequest.RootRole = *unleash.NewNullableFloat32(&roleID)
	}

	// Optional: MappingsSSO
	if !plan.MappingsSSO.IsNull() && len(plan.MappingsSSO.Elements()) > 0 {
		var mappings []string
		resp.Diagnostics.Append(plan.MappingsSSO.ElementsAs(ctx, &mappings, false)...)
		if !resp.Diagnostics.HasError() {
			updateGroupRequest.MappingsSSO = mappings
		}
	}

	// Optional: Users
	if !plan.Users.IsNull() && len(plan.Users.Elements()) > 0 {
		usersAPI := convertModelUsersToAPI(ctx, plan.Users, &resp.Diagnostics)
		if !resp.Diagnostics.HasError() {
			updateGroupRequest.Users = usersAPI
		}
	}

	// Execute API call
	group, apiResponse, err := r.client.UsersAPI.UpdateGroup(ctx, state.ID.ValueString()).CreateGroupSchema(updateGroupRequest).Execute()
	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}
	groupID := fmt.Sprint(*group.Id)

	// NOTE: Update does not return the user list as specified in the API spec, we need a Read to obtain the users.
	updatedGroup, httpResp, err := r.client.UsersAPI.GetGroup(ctx, groupID).Execute()
	// Homemade error checking for Read request in Update
	if err != nil || httpResp.StatusCode != 200 {
		return
	}
	// Populate the new state
	var newState groupResourceModel

	// Populate state from API response
	populateGroupStateFromAPI(ctx, updatedGroup, &newState, &resp.Diagnostics)

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
	tflog.Debug(ctx, "Finished updating group resource", map[string]any{"success": true})
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete group resource")

	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Delete the group
	httpResp, err := r.client.UsersAPI.DeleteGroup(ctx, state.ID.ValueString()).Execute()
	if !ValidateApiResponse(httpResp, 200, &resp.Diagnostics, err) {
		return
	}

	tflog.Debug(ctx, "Finished deleting group resource", map[string]any{"success": true})
}
