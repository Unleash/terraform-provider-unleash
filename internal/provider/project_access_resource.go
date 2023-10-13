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
	Users  types.List 	 `tfsdk:"users"`
	Groups types.List 	 `tfsdk:"groups"`
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
	var state projectAccessResourceModel

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished importing projectAccess data source", map[string]any{"success": true})
}

func (r *projectAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create projectAccess resource")
	var plan projectAccessResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var projectId = *plan.Project.ValueStringPointer()
	roles := []unleash.ProjectAccessConfigurationSchemaRolesInner{}
	for _, r := range plan.Roles {
		rolePayload := *unleash.NewProjectAccessConfigurationSchemaRolesInnerWithDefaults()
		roleId := int32(r.Role.ValueInt64())
		rolePayload.Id = &roleId
		tflog.Debug(ctx, fmt.Sprintf("Creating role %v", roleId))
		if !r.Users.IsNull() && !r.Users.IsUnknown() {
			tflog.Debug(ctx, fmt.Sprintf("Adding users %v", r.Users))
			resp.Diagnostics.Append(r.Users.ElementsAs(ctx, &rolePayload.Users, false)...)
		}

		if resp.Diagnostics.HasError() {
			return
		}

		tflog.Debug(ctx, fmt.Sprintf("Adding Groups %v", r.Groups))
		if !r.Groups.IsNull() && !r.Groups.IsUnknown() {
			resp.Diagnostics.Append(r.Groups.ElementsAs(ctx, &rolePayload.Groups, false)...)
		} else {
			tflog.Debug(ctx, fmt.Sprintf("None found %v", r.Groups))
			rolePayload.Groups = make([]int32, 0)
		}
		if resp.Diagnostics.HasError() {
			return
		}
		roles = append(roles, rolePayload)
	}
	accessConfiguration := *unleash.NewProjectAccessConfigurationSchema(roles)

	api_response, err := r.client.ProjectsAPI.SetProjectAccess(context.Background(), projectId).ProjectAccessConfigurationSchema(accessConfiguration).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read projectAccess",
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

	_, api_response, err := r.client.ProjectsAPI.GetProjectAccess(context.Background(), projectId).Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to Read ProjectAccess %s", state.Project.ValueString()),
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


	// Create a struct that represents a map by roleId having users and groups as lists of the struct
	// unleash.ProjectAccessConfigurationSchemaRolesInner
	// This is used to create a list of roles with their users and groups
	// rolesMap := make(map[int64]unleash.ProjectAccessConfigurationSchemaRolesInner)
	// for _, role := range projectAccess.Roles {
	// 	createdRole := rolesMap[int64(role.Id)]
	// 	if (createdRole == nil) {

	// 	}
	// 	role.Users, _ = basetypes.NewListValueFrom(ctx, types.StringType, role.Users)
	// }

	// state.Roles, _ = basetypes.NewListValueFrom(ctx, types.StringType, projectAccess.Roles)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading projectAccess data source", map[string]any{"success": true})
}

func (r *projectAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update projectAccess resource")
	var state projectAccessResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// updateProjectAccessSchema := *unleash.NewUpdateProjectAccessSchemaWithDefaults()
	// updateProjectAccessSchema.Name = *state.Name.ValueStringPointer()
	// if !state.Description.IsNull() {
	// 	updateProjectAccessSchema.Description = state.Description.ValueStringPointer()
	// }

	// req.State.Get(ctx, &state)

	// api_response, err := r.client.ProjectAccesssAPI.UpdateProjectAccess(context.Background(), state.Id.ValueString()).UpdateProjectAccessSchema(updateProjectAccessSchema).Execute()

	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Unable to update projectAccess ",
	// 		err.Error(),
	// 	)
	// 	return
	// }

	// if api_response.StatusCode != 200 {
	// 	resp.Diagnostics.AddError(
	// 		"Unexpected HTTP error code received",
	// 		api_response.Status,
	// 	)
	// 	return
	// }

	// // our update doesn't return the projectAccess, so we need to re-read it
	// projectAccesss, api_response, err := r.client.ProjectAccesssAPI.GetProjectAccesss(context.Background()).Execute()

	// var projectAccess unleash.ProjectAccessSchema
	// for _, p := range projectAccesss.ProjectAccesss {
	// 	if p.Id == state.Id.ValueString() {
	// 		projectAccess = p
	// 	}
	// }
	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		fmt.Sprintf("Unable to Read ProjectAccess %s", state.Id.ValueString()),
	// 		err.Error(),
	// 	)
	// 	return
	// }

	// if api_response.StatusCode != 200 {
	// 	resp.Diagnostics.AddError(
	// 		"Unexpected HTTP error code received",
	// 		api_response.Status,
	// 	)
	// 	return
	// }

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished updating projectAccess data source", map[string]any{"success": true})
}

func (r *projectAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete projectAccess")
	// var state projectAccessResourceModel
	// diags := req.State.Get(ctx, &state)
	// resp.Diagnostics.Append(diags...)

	// if resp.Diagnostics.HasError() {
	// 	return
	// }

	// api_response, err := r.client.ProjectAccesssAPI.DeleteProjectAccess(ctx, state.Id.ValueString()).Execute()

	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Unable to read projectAccess ",
	// 		err.Error(),
	// 	)
	// 	return
	// }

	// if api_response.StatusCode != 200 {
	// 	resp.Diagnostics.AddError(
	// 		"Unexpected HTTP error code received",
	// 		api_response.Status,
	// 	)
	// 	return
	// }

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, "Deleted item resource", map[string]any{"success": true})
}
