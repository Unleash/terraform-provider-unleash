package provider

import (
	"context"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &environmentDataSource{}
	_ datasource.DataSourceWithConfigure = &environmentDataSource{}
)

func NewEnvironmentDataSource() datasource.DataSource {
	return &environmentDataSource{}
}

type environmentDataSource struct {
	client *unleash.APIClient
}

type environmentDataSourceModel struct {
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (d *environmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		return
	}
	d.client = client
}

func (d *environmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *environmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a context field.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the environment. Must be a URL-friendly string according to RFC 3968.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the environment. Unleash recognizes 'development', 'test', 'preproduction' and 'production'. " +
					"You can pass other values and Unleash will accept them but they will carry no special semantics.",
				Required: true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether this environment is enabled or not. Enabled environments will be available for use in Feature Flags. This defaults to true if not set.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (d *environmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to hydrate environment")
	var state environmentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Unable to read environment data source")
		return
	}

	environment, apiResponse, err := d.client.EnvironmentsAPI.GetEnvironment(ctx, state.Name.ValueString()).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	state = environmentDataSourceModel{
		Name:    types.StringValue(environment.Name),
		Type:    types.StringValue(environment.Type),
		Enabled: types.BoolValue(environment.Enabled),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading environment field data source", map[string]any{"success": true})
}
