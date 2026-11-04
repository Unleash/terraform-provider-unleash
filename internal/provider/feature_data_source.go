package provider

import (
	"context"
	"net/http"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &featureDataSource{}
	_ datasource.DataSourceWithConfigure = &featureDataSource{}
)

func NewFeatureDataSource() datasource.DataSource {
	return &featureDataSource{}
}

type featureDataSource struct {
	client *unleash.APIClient
}

type featureDataSourceModel struct {
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Project        types.String `tfsdk:"project"`
	Type           types.String `tfsdk:"type"`
	Description    types.String `tfsdk:"description"`
	ImpressionData types.Bool   `tfsdk:"impression_data"`
	Stale          types.Bool   `tfsdk:"stale"`
	Environments   types.Map    `tfsdk:"environments"`
}

func (d *featureDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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

func (d *featureDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature"
}

func (d *featureDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a feature flag, including whether it is enabled in each environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The name of the feature flag.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the feature flag.",
				Required:    true,
			},
			"project": schema.StringAttribute{
				Description: "The project the feature flag belongs to. Defaults to `default`.",
				Optional:    true,
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the feature flag.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the feature flag.",
				Computed:    true,
			},
			"impression_data": schema.BoolAttribute{
				Description: "Whether impression data is emitted for this flag.",
				Computed:    true,
			},
			"stale": schema.BoolAttribute{
				Description: "Whether the flag is marked as stale.",
				Computed:    true,
			},
			"environments": schema.MapAttribute{
				Description: "Whether the flag is enabled, keyed by environment name. " +
					"For example, `data.unleash_feature.example.environments[\"production\"]`.",
				ElementType: types.BoolType,
				Computed:    true,
			},
		},
	}
}

func (d *featureDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read feature data source")
	var state featureDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := state.Project.ValueString()
	if project == "" {
		project = defaultFeatureProject
	}

	feature, httpRes, err := featureAPI{client: d.client}.get(ctx, project, state.Name.ValueString())
	if !ValidateApiResponse(httpRes, http.StatusOK, &resp.Diagnostics, err) {
		return
	}

	environments := make(map[string]bool, len(feature.Environments))
	for _, env := range feature.Environments {
		environments[env.Name] = env.Enabled
	}
	envMap, diags := types.MapValueFrom(ctx, types.BoolType, environments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Id = types.StringValue(feature.Name)
	state.Name = types.StringValue(feature.Name)
	state.Project = types.StringValue(feature.Project)
	state.Type = types.StringValue(feature.Type)
	state.ImpressionData = types.BoolValue(feature.ImpressionData)
	state.Stale = types.BoolValue(feature.Stale)
	state.Environments = envMap
	if feature.Description == nil || *feature.Description == "" {
		state.Description = types.StringNull()
	} else {
		state.Description = types.StringValue(*feature.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading feature data source", map[string]interface{}{"success": true})
}
