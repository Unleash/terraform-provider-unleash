package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &projectDataSource{}
	_ datasource.DataSourceWithConfigure = &projectDataSource{}
)

func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

type projectDataSource struct {
	client *unleash.APIClient
}

type projectDataSourceModel struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Mode        types.String `tfsdk:"mode"`
}

func (d *projectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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

func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a project definition.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of this project.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The id of this project.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the project's purpose.",
				Optional:    true,
			},
			"mode": schema.StringAttribute{
				Description: "The mode of the project affecting what actions are possible in this project. Possible values are 'open', 'protected', 'private'. Defaults to 'open' if not set.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read project data source")
	var state projectDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	projects, api_response, err := d.client.ProjectsAPI.GetProjects(ctx).Execute()

	if !ValidateApiResponse(api_response, 200, &resp.Diagnostics, err) {
		return
	}

	var project unleash.ProjectSchema
	for _, p := range projects.Projects {
		if p.Id == state.Id.ValueString() {
			project = p
		}
	}

	state = projectDataSourceModel{
		Id:   types.StringValue(fmt.Sprintf("%v", project.Id)),
		Name: types.StringValue(fmt.Sprintf("%v", project.Name)),
	}

	if project.Description.IsSet() && project.Description.Get() != nil {
		state.Description = types.StringValue(fmt.Sprintf("%v", *project.Description.Get()))
	} else {
		state.Description = types.StringNull()
	}

	if project.Mode != nil {
		state.Mode = types.StringValue(*project.Mode)
	} else {
		// From checking the API spec I don't believe this actually can happen but this gives us a nice
		// chance to have some backwards compatibility with older versions of the API where open was the only mode
		state.Mode = types.StringValue("open")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading user data source", map[string]any{"success": true})
}
