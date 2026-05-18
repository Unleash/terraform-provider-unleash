package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	datasourcevalidator "github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &groupDataSource{}
	_ datasource.DataSourceWithConfigure = &groupDataSource{}
)

func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

type groupDataSource struct {
	client *unleash.APIClient
}

func (d *groupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	d.client = client
}

func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("name"),
		),
	}
}

func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a group by id or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this group",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name for this group",
				Optional:    true,
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the group's purpose.",
				Computed:    true,
			},
			"mappings_sso": schema.ListAttribute{
				Description: "SSO group mappings for this group.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"root_role": schema.Int64Attribute{
				Description: "The root role ID for this group.",
				Computed:    true,
			},
			"users": schema.ListAttribute{
				Description: "List of user IDs in this group.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
		},
	}
}

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read group data source")

	var state groupResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	lookup, ok := validateGroupLookup(state, &resp.Diagnostics)
	if !ok {
		return
	}

	group, ok := d.fetchGroup(ctx, lookup, resp)
	if !ok {
		return
	}

	populateGroupStateFromAPI(ctx, group, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading group data source", map[string]any{"success": true})
}

type groupLookupInput struct {
	id           string
	name         string
	idProvided   bool
	nameProvided bool
}

func validateGroupLookup(config groupResourceModel, diagnostics *diag.Diagnostics) (groupLookupInput, bool) {
	lookup := groupLookupInput{}

	if config.ID.IsUnknown() {
		diagnostics.AddError("Cannot use unknown value for id", "Provide a concrete id or omit it in favour of the name lookup.")
		return lookup, false
	}
	if config.Name.IsUnknown() {
		diagnostics.AddError("Cannot use unknown value for name", "Provide a concrete name or omit it in favour of the id lookup.")
		return lookup, false
	}

	if !config.ID.IsNull() {
		lookup.id = config.ID.ValueString()
		lookup.idProvided = lookup.id != ""
	}
	if !config.Name.IsNull() {
		lookup.name = config.Name.ValueString()
		lookup.nameProvided = lookup.name != ""
	}

	if !lookup.idProvided && !lookup.nameProvided {
		diagnostics.AddError("Missing group lookup value", "Provide a non-empty id or name.")
		return lookup, false
	}

	return lookup, true
}

func (d *groupDataSource) fetchGroup(ctx context.Context, lookup groupLookupInput, resp *datasource.ReadResponse) (*unleash.GroupSchema, bool) {
	if lookup.idProvided {
		return d.fetchGroupByID(ctx, lookup, resp)
	}
	return d.fetchGroupByName(ctx, lookup.name, resp)
}

func (d *groupDataSource) fetchGroupByID(ctx context.Context, lookup groupLookupInput, resp *datasource.ReadResponse) (*unleash.GroupSchema, bool) {
	group, apiResponse, err := d.client.UsersAPI.GetGroup(ctx, lookup.id).Execute()
	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return nil, false
	}

	if lookup.nameProvided && group.Name != lookup.name {
		resp.Diagnostics.AddError(
			"Group id and name mismatch",
			fmt.Sprintf("Group %s has name %q, which does not match the requested name %q.", lookup.id, group.Name, lookup.name),
		)
		return nil, false
	}

	return group, true
}

func (d *groupDataSource) fetchGroupByName(ctx context.Context, name string, resp *datasource.ReadResponse) (*unleash.GroupSchema, bool) {
	groups, apiResponse, err := d.client.UsersAPI.GetGroups(ctx).Execute()
	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return nil, false
	}
	if groups == nil {
		resp.Diagnostics.AddError("Nil Groups Response", "The API returned a nil groups response.")
		return nil, false
	}

	var matched *unleash.GroupSchema
	for i := range groups.Groups {
		if groups.Groups[i].Name != name {
			continue
		}
		if matched != nil {
			resp.Diagnostics.AddError(
				"Multiple groups found",
				fmt.Sprintf("More than one group matched the name %q. Use id to select a specific group.", name),
			)
			return nil, false
		}
		matched = &groups.Groups[i]
	}

	if matched == nil {
		resp.Diagnostics.AddError(
			"Group not found",
			fmt.Sprintf("No group matched the name %q.", name),
		)
		return nil, false
	}

	if matched.Id == nil {
		resp.Diagnostics.AddError("Nil Group ID Response", "The API returned a matching group without an id.")
		return nil, false
	}

	group, apiResponse, err := d.client.UsersAPI.GetGroup(ctx, fmt.Sprint(*matched.Id)).Execute()
	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return nil, false
	}

	return group, true
}
