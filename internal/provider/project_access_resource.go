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

var (
	_ resource.Resource                = &projectAccessResource{}
	_ resource.ResourceWithConfigure   = &projectAccessResource{}
	_ resource.ResourceWithImportState = &projectAccessResource{}
)

func NewProjectAccessResource() resource.Resource {
	return &projectAccessResource{}
}

type projectAccessResource struct {
	client *unleash.APIClient
}

type roleWithMembers struct {
	Role   types.Int64   `tfsdk:"role"`
	Users  []types.Int64 `tfsdk:"users"`
	Groups []types.Int64 `tfsdk:"groups"`
}

type projectAccessResourceModel struct {
	Project types.String      `tfsdk:"project"`
	Roles   []roleWithMembers `tfsdk:"roles"`
}

// Configure adds the provider configured client to the data source.
func (r *projectAccessResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
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
func (r *projectAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_access"
}

// Schema defines the schema for the data source. TODO: can we transform OpenAPI schema into TF schema?
func (r *projectAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "ProjectAccess schema",
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{
				Description: "Project identifier.",
				Required:    true,
			},
			"roles": schema.ListNestedAttribute{
				Description: "Roles available in this project with their members.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role": schema.Int64Attribute{
							Description: "The role identifier.",
							Required:    true,
						},
						"users": schema.ListAttribute{
							Description: "List of users with this role assigned.",
							Required:    true,
							ElementType: types.Int64Type,
						},
						"groups": schema.ListAttribute{
							Description: "List of projects with this role assigned.",
							Required:    true,
							ElementType: types.Int64Type,
						},
					},
				},
			},
		},
	}
}

func (r *projectAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import projectAccess resource")

	resource.ImportStatePassthroughID(ctx, path.Root("project"), req, resp)

	tflog.Debug(ctx, "Finished importing projectAccess data source", map[string]any{"success": true})
}

func (r *projectAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create projectAccess resource")
	var plan projectAccessResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Upserting %v", plan))
	resp.Diagnostics.Append(r.upsertProjectAccess(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished creating projectAccess data source", map[string]any{"success": true})
}

func (r *projectAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read projectAccess resource")
	var state projectAccessResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectId := *state.Project.ValueStringPointer()

	projectAccess, api_response, err := r.client.ProjectsAPI.GetProjectAccess(ctx, projectId).Execute()

	if (!ExpectedResponse(api_response, 200, &resp.Diagnostics, err)) {
		return
	}

	state.Roles = transformToInternalRoles(projectAccess)

	tflog.Info(ctx, fmt.Sprintf("Read projectAccess %v", state))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading projectAccess data source", map[string]any{"success": true})
}

func (r *projectAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update project access resource")
	var plan projectAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.upsertProjectAccess(ctx, plan)...)

	var state projectAccessResource
	req.State.Get(ctx, &state)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "Finished updating projectAccess data source", map[string]any{"success": true})
}

func (r *projectAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Resource Not Deleted", "The projectAccess resource was removed from the Terraform state, but not deleted from the actual system. This is to avoid potential mistakes. Instead of deleting projectAccess you may just delete the whole project")
}

/** Helper methods **/

func (r *projectAccessResource) upsertProjectAccess(ctx context.Context, plan projectAccessResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	var projectId = plan.Project.ValueString()
	roles := []unleash.ProjectAccessConfigurationSchemaRolesInner{}
	for _, r := range plan.Roles {
		rolePayload := *unleash.NewProjectAccessConfigurationSchemaRolesInnerWithDefaults()
		roleId := int32(r.Role.ValueInt64())
		rolePayload.Id = &roleId
		tflog.Debug(ctx, fmt.Sprintf("Creating role %v", roleId))
		// Handle users for this role
		for _, user := range r.Users {
			rolePayload.Users = append(rolePayload.Users, int32(user.ValueInt64()))
		}
		tflog.Debug(ctx, fmt.Sprintf("Added users: %v to role %v", rolePayload.Users, roleId))

		// Handle groups for this role
		for _, group := range r.Groups {
			rolePayload.Groups = append(rolePayload.Groups, int32(group.ValueInt64()))
		}
		tflog.Debug(ctx, fmt.Sprintf("Added groups: %v to role %v", rolePayload.Groups, roleId))

		if diagnostics.HasError() {
			return diagnostics
		}
		roles = append(roles, rolePayload)
	}
	accessConfiguration := *unleash.NewProjectAccessConfigurationSchema(roles)

	api_response, err := r.client.ProjectsAPI.SetProjectAccess(ctx, projectId).ProjectAccessConfigurationSchema(accessConfiguration).Execute()

	ExpectedResponse(api_response, 200, &diagnostics, err)

	return diagnostics
}

func transformToInternalRoles(accessSchema *unleash.ProjectAccessSchema) []roleWithMembers {
	var internalRoles []roleWithMembers

	// Create a map for users and groups for efficient lookup
	usersMap := make(map[int32][]int32)  // Role ID to User IDs
	groupsMap := make(map[int32][]int32) // Role ID to Group IDs

	for _, user := range accessSchema.Users {
		for _, roleId := range user.Roles {
			usersMap[roleId] = append(usersMap[roleId], user.Id)
		}
	}

	for _, group := range accessSchema.Groups {
		for _, roleId := range group.Roles {
			groupsMap[roleId] = append(groupsMap[roleId], group.Id)
		}
	}

	// Populate the internalRoles slice
	for _, role := range accessSchema.Roles {
		internalRole := roleWithMembers{
			Role:   types.Int64Value(int64(role.Id)),
			Users:  []types.Int64{},
			Groups: []types.Int64{},
		}

		// Add users to the role
		for _, userId := range usersMap[role.Id] {
			internalRole.Users = append(internalRole.Users, types.Int64Value(int64(userId)))
		}

		// Add groups to the role
		for _, groupId := range groupsMap[role.Id] {
			internalRole.Groups = append(internalRole.Groups, types.Int64Value(int64(groupId)))
		}

		// if both roles are empty, don't keep the role in the state
		if (len(internalRole.Users) > 0) || (len(internalRole.Groups) > 0) {
			internalRoles = append(internalRoles, internalRole)
		}
	}

	return internalRoles
}
