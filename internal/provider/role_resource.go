package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithConfigure   = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

// NewRoleResource is a helper function to simplify the provider implementation.
func NewRoleResource() resource.Resource {
	return &roleResource{}
}

// roleResource is the resource implementation.
type roleResource struct {
	client *unleash.APIClient
}

type permissionRef struct {
	Name        types.String `tfsdk:"name"`
	Environment types.String `tfsdk:"environment"`
}

type roleResourceModel struct {
	Id          types.String    `tfsdk:"id"`
	Name        types.String    `tfsdk:"name"`
	Type        types.String    `tfsdk:"type"`
	Description types.String    `tfsdk:"description"`
	Permissions []permissionRef `tfsdk:"permissions"`
}

// Configure adds the provider configured client to the resource.
func (r *roleResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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
func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the resource. TODO: can we transform OpenAPI schema into TF schema?
func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Role schema",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of this role.",
				Required:    true,
			},
			"id": schema.StringAttribute{
				Description: "The id of this role.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "A role can either be a global root role (applies to all roles) or a role role.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A more detailed description of the role and what use it's intended for.",
				Required:    true,
			},
			"permissions": schema.ListNestedAttribute{
				Description: "A more detailed description of the role and what use it's intended for.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of this permission.",
							Required:    true,
						},
						"environment": schema.StringAttribute{
							Description: "For which environment this permission applies (note that only environment-type permissions can have an environment).",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import role resource")

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Debug(ctx, "Finished importing role resource", map[string]any{"success": true})
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create role resource")
	var plan roleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleWithPermissions := *unleash.NewCreateRoleWithPermissionsSchemaAnyOf(plan.Name.ValueString())
	roleWithPermissions.Type = plan.Type.ValueStringPointer()
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		roleWithPermissions.Description = plan.Description.ValueStringPointer()
	}

	permissions := make([]unleash.CreateRoleWithPermissionsSchemaAnyOfPermissionsInner, 0, len(plan.Permissions))
	for _, plannedPermission := range plan.Permissions {
		permissionRef := unleash.CreateRoleWithPermissionsSchemaAnyOfPermissionsInner{
			Name: plannedPermission.Name.ValueString(),
		}
		if !plannedPermission.Environment.IsNull() && !plannedPermission.Environment.IsUnknown() {
			permissionRef.Environment = plannedPermission.Environment.ValueStringPointer()
		}
		permissions = append(permissions, permissionRef)
	}
	roleWithPermissions.Permissions = permissions

	createRoleRequest := unleash.CreateRoleWithPermissionsSchema{}
	createRoleRequest.CreateRoleWithPermissionsSchemaAnyOf = &roleWithPermissions

	role, api_response, err := r.client.UsersAPI.CreateRole(ctx).CreateRoleWithPermissionsSchema(createRoleRequest).Execute()

	if !ExpectedResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	// Update model with response
	createdRole := role.Roles
	tflog.Debug(ctx, fmt.Sprintf("Created role: %+v", createdRole))
	newState := roleResourceModel{
		Id:          types.StringValue(fmt.Sprintf("%v", createdRole.Id)),
		Name:        types.StringValue(createdRole.Name),
		Type:        types.StringValue(createdRole.Type),
		Description: types.StringValue(*role.Roles.Description),
	}
	if createdRole.Description != nil {
		newState.Description = types.StringValue(*createdRole.Description)
	} else {
		newState.Description = types.StringNull()
	}
	// response does not include permissions, so we'll use the ones sent in the request
	newPermissions := make([]permissionRef, 0, len(permissions))
	for _, requestPermission := range permissions {
		permissionRef := permissionRef{
			Name: types.StringValue(requestPermission.Name),
		}
		if requestPermission.Environment != nil {
			permissionRef.Environment = types.StringValue(*requestPermission.Environment)
		}
		newPermissions = append(newPermissions, permissionRef)
	}
	newState.Permissions = newPermissions

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
	tflog.Debug(ctx, "Finished creating role resource", map[string]any{"success": true})
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read role resource")
	var state roleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	roleId := state.Id.ValueString()
	role, api_response, err := r.client.UsersAPI.GetRoleById(ctx, roleId).Execute()

	if !ExpectedResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	state = roleResourceModel{
		Id:   types.StringValue(fmt.Sprintf("%v", role.Id)),
		Name: types.StringValue(role.Name),
		Type: types.StringValue(role.Type),
	}

	if role.Description != nil {
		state.Description = types.StringValue(*role.Description)
	} else {
		state.Description = types.StringNull()
	}

	permissions := make([]permissionRef, 0, len(role.Permissions))
	for _, foundPermissions := range role.Permissions {
		permissionRef := permissionRef{
			Name: types.StringValue(foundPermissions.Name),
		}
		if foundPermissions.Environment != nil {
			permissionRef.Environment = types.StringValue(*foundPermissions.Environment)
		}
		permissions = append(permissions, permissionRef)
	}
	state.Permissions = permissions
	tflog.Debug(ctx, fmt.Sprintf("Reading permissions: %+v into %+v", role.Permissions, permissions))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading role resource", map[string]any{"success": true})
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update role resource")
	var state roleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	roleWithPermissions := *unleash.NewCreateRoleWithPermissionsSchemaAnyOf(state.Name.ValueString())
	roleWithPermissions.Type = state.Type.ValueStringPointer()
	if !state.Description.IsNull() && !state.Description.IsUnknown() {
		roleWithPermissions.Description = state.Description.ValueStringPointer()
	}

	permissions := make([]unleash.CreateRoleWithPermissionsSchemaAnyOfPermissionsInner, 0, len(state.Permissions))
	for _, plannedPermission := range state.Permissions {
		permissionRef := unleash.CreateRoleWithPermissionsSchemaAnyOfPermissionsInner{
			Name: plannedPermission.Name.ValueString(),
		}
		if !plannedPermission.Environment.IsNull() && !plannedPermission.Environment.IsUnknown() {
			permissionRef.Environment = plannedPermission.Environment.ValueStringPointer()
		}
		permissions = append(permissions, permissionRef)
	}
	roleWithPermissions.Permissions = permissions

	updateRoleSchema := unleash.CreateRoleWithPermissionsSchema{}
	updateRoleSchema.CreateRoleWithPermissionsSchemaAnyOf = &roleWithPermissions

	req.State.Get(ctx, &state) // the id is part of the state, not the plan, this is how we get its value
	roleId := state.Id.ValueString()
	roleWithVersion, api_response, err := r.client.UsersAPI.UpdateRole(ctx, roleId).CreateRoleWithPermissionsSchema(updateRoleSchema).Execute()

	if !ExpectedResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	role := roleWithVersion.Roles
	state = roleResourceModel{
		Id:   types.StringValue(fmt.Sprintf("%v", role.Id)),
		Name: types.StringValue(role.Name),
		Type: types.StringValue(role.Type),
	}

	if role.Description != nil {
		state.Description = types.StringValue(*role.Description)
	} else {
		state.Description = types.StringNull()
	}

	freshPermissions := make([]permissionRef, 0, len(permissions))
	for _, foundPermissions := range permissions {
		permissionRef := permissionRef{
			Name: types.StringValue(foundPermissions.Name),
		}
		if foundPermissions.Environment != nil {
			permissionRef.Environment = types.StringValue(*foundPermissions.Environment)
		}
		freshPermissions = append(freshPermissions, permissionRef)
	}
	state.Permissions = freshPermissions

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished updating role resource", map[string]any{"success": true})
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete role")
	var state roleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	api_response, err := r.client.UsersAPI.DeleteRole(ctx, state.Id.ValueString()).Execute()

	if !ExpectedResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, "Deleted role resource", map[string]any{"success": true})
}
