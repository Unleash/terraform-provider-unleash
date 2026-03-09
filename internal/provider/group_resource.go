package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
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

func emptyStringList() types.List {
	return types.ListValueMust(types.StringType, []attr.Value{})
}

func emptyInt64List() types.List {
	return types.ListValueMust(types.Int64Type, []attr.Value{})
}

func stringListStateValue(ctx context.Context, current types.List, values []string, diagnostics *diag.Diagnostics) types.List {
	if len(values) > 0 {
		listValue, diags := types.ListValueFrom(ctx, types.StringType, values)
		diagnostics.Append(diags...)
		return listValue
	}
	if current.IsNull() {
		return types.ListNull(types.StringType)
	}
	return emptyStringList()
}

func int64ListStateValue(ctx context.Context, current types.List, values []int64, diagnostics *diag.Diagnostics) types.List {
	if len(values) > 0 {
		listValue, diags := types.ListValueFrom(ctx, types.Int64Type, values)
		diagnostics.Append(diags...)
		return listValue
	}
	if current.IsNull() {
		return types.ListNull(types.Int64Type)
	}
	return emptyInt64List()
}

func setGroupRequestFromPlan(ctx context.Context, request *unleash.CreateGroupSchema, plan groupResourceModel, diagnostics *diag.Diagnostics) {
	request.Name = plan.Name.ValueString()

	if !plan.Description.IsUnknown() {
		if plan.Description.IsNull() {
			request.Description = *unleash.NewNullableString(nil)
		} else {
			request.Description = *unleash.NewNullableString(plan.Description.ValueStringPointer())
		}
	}

	if !plan.RootRole.IsUnknown() {
		if plan.RootRole.IsNull() {
			request.RootRole = *unleash.NewNullableFloat32(nil)
		} else {
			roleID := float32(plan.RootRole.ValueInt64())
			request.RootRole = *unleash.NewNullableFloat32(&roleID)
		}
	}

	if !plan.MappingsSSO.IsUnknown() {
		if plan.MappingsSSO.IsNull() {
			request.MappingsSSO = []string{}
		} else {
			var mappings []string
			diagnostics.Append(plan.MappingsSSO.ElementsAs(ctx, &mappings, false)...)
			if !diagnostics.HasError() {
				request.MappingsSSO = mappings
			}
		}
	}

	if !plan.Users.IsUnknown() {
		if plan.Users.IsNull() {
			request.Users = []unleash.CreateGroupSchemaUsersInner{}
		} else {
			usersAPI := convertModelUsersToAPI(ctx, plan.Users, diagnostics)
			if !diagnostics.HasError() {
				request.Users = usersAPI
			}
		}
	}
}

// Helper function to populate group state from API response.
func populateGroupStateFromAPI(ctx context.Context, group *unleash.GroupSchema, state *groupResourceModel, diagnostics *diag.Diagnostics) {
	if group == nil {
		tflog.Error(ctx, "populateGroupStateFromAPI: group is nil")
		diagnostics.AddError("Nil Group Response", "The API returned a nil group response")
		return
	}
	// Check the ID
	if group.Id == nil {
		tflog.Error(ctx, "populateGroupStateFromAPI: group.Id is nil")
		diagnostics.AddError("Nil Group ID Response", "The API returned a nil group id response")
		return
	}
	// Set the ID
	state.ID = types.StringValue(fmt.Sprint(*group.Id))

	// Name
	state.Name = types.StringValue(group.Name)

	// Description
	if group.Description.IsSet() && group.Description.Get() != nil {
		state.Description = types.StringValue(*group.Description.Get())
	} else {
		state.Description = types.StringNull()
	}

	// RootRole.
	if group.RootRole.IsSet() && group.RootRole.Get() != nil {
		rolePtr := group.RootRole.Get()
		state.RootRole = types.Int64Value(int64(*rolePtr))
	} else {
		state.RootRole = types.Int64Null()
	}

	state.MappingsSSO = stringListStateValue(ctx, state.MappingsSSO, group.MappingsSSO, diagnostics)

	tflog.Debug(ctx, "populateGroupStateFromAPI: Processing users", map[string]any{
		"users_nil": group.Users == nil,
		"users_len": len(group.Users),
	})

	state.Users = int64ListStateValue(ctx, state.Users, convertAPIUsersToModel(group.Users), diagnostics)
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
	setGroupRequestFromPlan(ctx, &createGroupRequest, plan, &resp.Diagnostics)

	// Execute API call
	// NOTE: Create does not return the user list as specified in the API spec, we need a Read to obtain the users
	group, apiResponse, err := r.client.UsersAPI.CreateGroup(ctx).CreateGroupSchema(createGroupRequest).Execute()
	if !ValidateApiResponse(apiResponse, 201, &resp.Diagnostics, err) {
		return
	}
	if group == nil || group.Id == nil {
		resp.Diagnostics.AddError(
			"API Error: Missing Group ID",
			"The Unleash API successfully created the group, but did not return a Group ID in the response. "+
				"This prevents Terraform from tracking the resource.",
		)
		return
	}

	groupID := fmt.Sprint(*group.Id)
	// Read to get the group
	createdGroup, apiReadResponse, err := r.client.UsersAPI.GetGroup(ctx, groupID).Execute()
	// Validate Read response in Create flow so failures surface diagnostics.
	if !ValidateApiResponse(apiReadResponse, 200, &resp.Diagnostics, err) {
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
	setGroupRequestFromPlan(ctx, &updateGroupRequest, plan, &resp.Diagnostics)

	// Execute API call
	group, apiResponse, err := r.client.UsersAPI.UpdateGroup(ctx, state.ID.ValueString()).CreateGroupSchema(updateGroupRequest).Execute()
	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}
	if group == nil || group.Id == nil {
		resp.Diagnostics.AddError(
			"API Error: Missing Group ID",
			"The Unleash API successfully updated the group, but did not return a Group ID in the response. "+
				"This prevents Terraform from tracking the resource.",
		)
		return
	}
	groupID := fmt.Sprint(*group.Id)

	// NOTE: Update does not return the user list as specified in the API spec, we need a Read to obtain the users.
	updatedGroup, apiReadResponse, err := r.client.UsersAPI.GetGroup(ctx, groupID).Execute()
	// Validate Read response in Update flow so failures surface diagnostics.
	if !ValidateApiResponse(apiReadResponse, 200, &resp.Diagnostics, err) {
		return
	}
	// Populate the new state
	newState := plan

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
	resp.State.RemoveResource(ctx)
}
