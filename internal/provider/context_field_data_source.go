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
	_ datasource.DataSource              = &contextFieldDataSource{}
	_ datasource.DataSourceWithConfigure = &contextFieldDataSource{}
)

func NewContextFieldDataSource() datasource.DataSource {
	return &contextFieldDataSource{}
}

type contextFieldDataSource struct {
	client *unleash.APIClient
}

type legalValueResourceModel struct {
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
}

type contextFieldDataSourceModel struct {
	Name        types.String              `tfsdk:"name"`
	Description types.String              `tfsdk:"description"`
	Stickiness  types.Bool                `tfsdk:"stickiness"`
	LegalValues []legalValueResourceModel `tfsdk:"legal_values"`
}

func (d *contextFieldDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
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

func (d *contextFieldDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context_field"
}

func (d *contextFieldDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a context field.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the context field.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the context field.",
				Optional:    true,
				Computed:    true,
			},
			"stickiness": schema.BoolAttribute{
				Description: "Whether this field is available for custom stickiness",
				Optional:    true,
				Computed:    true,
			},
			"legal_values": schema.ListNestedAttribute{
				Description: "Legal values for this context field. If not set, then any value is available for this context field.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Description: "The allowed value.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the allowed value.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *contextFieldDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to hydrate context field")
	var state contextFieldDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Unable to read context field data source")
		return
	}

	var contextField, apiResponse, err = d.client.ContextAPI.GetContextField(ctx, state.Name.ValueString()).Execute()

	if !ValidateApiResponse(apiResponse, 200, &resp.Diagnostics, err) {
		return
	}

	legalValues := make([]legalValueResourceModel, len(contextField.LegalValues))
	for i, legalValue := range contextField.LegalValues {
		legalValues[i] = legalValueResourceModel{
			Value: types.StringValue(legalValue.Value),
		}

		if legalValue.Description != nil {
			legalValues[i].Description = types.StringValue(*legalValue.Description)
		} else {
			legalValues[i].Description = types.StringNull()
		}
	}

	state = contextFieldDataSourceModel{
		Name:        types.StringValue(contextField.Name),
		LegalValues: legalValues,
	}

	if contextField.Description.IsSet() {
		state.Description = types.StringValue(*contextField.Description.Get())
	} else {
		state.Description = types.StringNull()
	}

	if contextField.Stickiness != nil {
		state.Stickiness = types.BoolValue(*contextField.Stickiness)
	} else {
		state.Stickiness = types.BoolNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished reading context field data source", map[string]any{"success": true})
}
