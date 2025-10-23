package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	datasourcevalidator "github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

// NewUserDataSource is a helper function to simplify the provider implementation.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource is the data source implementation.
type userDataSource struct {
	client *unleash.APIClient
}

type userDataSourceModel struct {
	Id       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Email    types.String `tfsdk:"email"`
	Name     types.String `tfsdk:"name"`
	RootRole types.Int64  `tfsdk:"root_role"`
}

// Configure adds the provider configured client to the data source.
func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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

// Metadata returns the data source type name.
func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the data source. TODO: can we transform OpenAPI schema into TF schema?
func (d *userDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("id"),
			path.MatchRoot("email"),
		),
	}
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a user by id or email.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this user.",
				Optional:    true,
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email of the user.",
				Computed:    true,
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username of the user.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the user.",
				Optional:    true,
				Computed:    true,
			},
			"root_role": schema.Int64Attribute{
				Description: "The role id for the user.",
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read user data source")
	var config userDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...) // capture user input
	if resp.Diagnostics.HasError() {
		return
	}

	lookup, ok := validateUserLookup(config, &resp.Diagnostics)
	if !ok {
		return
	}

	user, ok := d.fetchUser(ctx, lookup, resp)
	if !ok {
		return
	}

	state := buildUserState(user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...) // update state with fresh data
	tflog.Debug(ctx, "Finished reading user data source", map[string]any{"success": true})
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func emailMatches(actual *string, expected string) bool {
	return actual != nil && strings.EqualFold(*actual, expected)
}

type userLookupInput struct {
	id            string
	email         string
	idProvided    bool
	emailProvided bool
}

func validateUserLookup(config userDataSourceModel, diags *diag.Diagnostics) (userLookupInput, bool) {
	lookup := userLookupInput{}

	if config.Id.IsUnknown() {
		diags.AddError("Cannot use unknown value for id", "Provide a concrete id or omit it in favour of the email lookup.")
		return lookup, false
	}
	if config.Email.IsUnknown() {
		diags.AddError("Cannot use unknown value for email", "Provide a concrete email or omit it in favour of the id lookup.")
		return lookup, false
	}

	if !config.Id.IsNull() {
		lookup.id = config.Id.ValueString()
		lookup.idProvided = lookup.id != ""
	}
	if !config.Email.IsNull() {
		lookup.email = config.Email.ValueString()
		lookup.emailProvided = lookup.email != ""
	}

	return lookup, true
}

func (d *userDataSource) fetchUser(ctx context.Context, lookup userLookupInput, resp *datasource.ReadResponse) (*unleash.UserSchema, bool) {
	if lookup.idProvided {
		return d.fetchUserByID(ctx, lookup, resp)
	}
	return d.fetchUserByEmail(ctx, lookup.email, resp)
}

func (d *userDataSource) fetchUserByID(ctx context.Context, lookup userLookupInput, resp *datasource.ReadResponse) (*unleash.UserSchema, bool) {
	idValue, err := strconv.Atoi(lookup.id)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("User id was not a number %s", lookup.id),
			err.Error(),
		)
		return nil, false
	}

	apiUser, apiResponse, err := d.client.UsersAPI.GetUser(ctx, int32(idValue)).Execute()
	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return nil, false
	}

	if lookup.emailProvided && !emailMatches(apiUser.Email, lookup.email) {
		resp.Diagnostics.AddError(
			"User id and email mismatch",
			fmt.Sprintf("User %s has email %q, which does not match the requested email %q.", lookup.id, valueOrEmpty(apiUser.Email), lookup.email),
		)
		return nil, false
	}

	return apiUser, true
}

func (d *userDataSource) fetchUserByEmail(ctx context.Context, email string, resp *datasource.ReadResponse) (*unleash.UserSchema, bool) {
	searchResult, apiResponse, err := d.client.UsersAPI.SearchUsers(ctx).Q(email).Execute()
	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return nil, false
	}

	for i := range searchResult.Users {
		candidate := searchResult.Users[i]
		if candidate.Email != nil && strings.EqualFold(*candidate.Email, email) {
			detailedUser, apiResponse, err := d.client.UsersAPI.GetUser(ctx, candidate.Id).Execute()
			if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
				return nil, false
			}
			return detailedUser, true
		}
	}

	resp.Diagnostics.AddError(
		"User not found",
		fmt.Sprintf("No user matched the email %q.", email),
	)
	return nil, false
}

func buildUserState(user *unleash.UserSchema) userDataSourceModel {
	state := userDataSourceModel{
		Id: types.StringValue(strconv.Itoa(int(user.Id))),
	}

	if user.RootRole != nil {
		state.RootRole = types.Int64Value(int64(*user.RootRole))
	} else {
		state.RootRole = types.Int64Null()
	}

	if user.Username.IsSet() && user.Username.Get() != nil {
		state.Username = types.StringValue(*user.Username.Get())
	} else {
		state.Username = types.StringNull()
	}

	if user.Email != nil {
		state.Email = types.StringValue(*user.Email)
	} else {
		state.Email = types.StringNull()
	}

	if user.Name.IsSet() && user.Name.Get() != nil {
		state.Name = types.StringValue(*user.Name.Get())
	} else {
		state.Name = types.StringNull()
	}

	return state
}
