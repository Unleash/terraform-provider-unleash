package provider

import (
	"context"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &projectEnvironmentDataSource{}
	_ datasource.DataSourceWithConfigure = &projectEnvironmentDataSource{}
)

func NewProjectEnvironmentDataSource() datasource.DataSource {
	return &projectEnvironmentDataSource{}
}

type projectEnvironmentDataSource struct {
	client *unleash.APIClient
}

type projectEnvironmentDataSourceModel struct {
	ProjectId             types.String `tfsdk:"project_id"`
	EnvironmentName       types.String `tfsdk:"environment_name"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	ChangeRequestsEnabled types.Bool   `tfsdk:"change_requests_enabled"`
	RequiredApprovals     types.Int64  `tfsdk:"required_approvals"`
}

func (d *projectEnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		return
	}
	d.client = client
}

func (d *projectEnvironmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_environment"
}

func (d *projectEnvironmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "ProjectEnvironment schema",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "Project identifier.",
				Required:    true,
			},
			"environment_name": schema.StringAttribute{
				Description: "Environment identifier, equivalent to the environment name.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "If the environment is enabled for this project. This affects whether or not users will be able to enable flags for this environment on this project.",
				Computed:    true,
			},
			"change_requests_enabled": schema.BoolAttribute{
				Description: "If change requests are required for this environment, the environment must be enabled for this to have effect.",
				Computed:    true,
			},
			"required_approvals": schema.Int64Attribute{
				Description: "Number of approvals required for change requests.",
				Computed:    true,
			},
		},
	}
}

func (d *projectEnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read project environment change data source")

	var state projectEnvironmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	config, getResponse, getErr := d.client.ChangeRequestsAPI.GetProjectChangeRequestConfig(ctx, state.ProjectId.ValueString()).Execute()

	if !ValidateApiResponse(getResponse, 200, &resp.Diagnostics, getErr) {
		return
	}

	var envChangeRequestConfig *unleash.ChangeRequestEnvironmentConfigSchema

	for _, env := range config {
		if env.Environment == state.EnvironmentName.ValueString() {
			envChangeRequestConfig = &env
			break
		}
	}

	if envChangeRequestConfig == nil {
		state.ChangeRequestsEnabled = types.BoolValue(false)
		state.RequiredApprovals = types.Int64Null()
		state.Enabled = types.BoolValue(false)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	var requiredApprovals basetypes.Int64Value

	if envChangeRequestConfig.RequiredApprovals.IsSet() && envChangeRequestConfig.RequiredApprovals.Get() != nil {
		requiredApprovals = types.Int64Value(int64(*envChangeRequestConfig.RequiredApprovals.Get()))
	} else {
		requiredApprovals = types.Int64Null()
	}

	state.ProjectId = types.StringValue(state.ProjectId.ValueString())
	state.EnvironmentName = types.StringValue(state.EnvironmentName.ValueString())
	state.ChangeRequestsEnabled = types.BoolValue(envChangeRequestConfig.ChangeRequestEnabled)
	state.RequiredApprovals = requiredApprovals
	state.Enabled = types.BoolValue(true)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading project environment change request", map[string]interface{}{"success": true})
}
