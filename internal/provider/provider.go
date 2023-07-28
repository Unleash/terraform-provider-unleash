// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	unleash "github.com/Unleash/unleash-server-api-go/client"

	"github.com/fatih/structs"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &UnleashProvider{}

// ScaffoldingProvider defines the provider implementation.
type UnleashProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaffoldingProviderModel describes the provider data model.
type UnleashConfiguration struct {
	BaseUrl       types.String `tfsdk:"base_url"`
	Authorization types.String `tfsdk:"authorization"`
}

func (p *UnleashProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "unleash"
	resp.Version = p.version
}

func (p *UnleashProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Unleash base URL (everything before `/api`)",
				Required:            true,
			},
			"authorization": schema.StringAttribute{
				MarkdownDescription: "Authhorization token for Unleash API",
				Required:            true,
			},
		},
		Description: "Interface with Unleash server API.",
	}
}

func (p *UnleashProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data UnleashConfiguration

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }
	tflog.Info(ctx, "Configuring Unleash client", structs.Map(data))
	tflog.Info(ctx, "Base URL: "+data.BaseUrl.ValueString())
	configuration := unleash.NewConfiguration()
	configuration.Servers = unleash.ServerConfigurations{
		unleash.ServerConfiguration{
			URL:         data.BaseUrl.ValueString(),
			Description: "Unleash server",
		},
	}
	configuration.AddDefaultHeader("Authorization", data.Authorization.ValueString())

	configuration.HTTPClient = httpClient(p.version == "dev" || p.version == "test")
	client := unleash.NewAPIClient(configuration)

	// Make the Inventory client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
	tflog.Info(ctx, "Configured Unleash client", map[string]any{"success": true})
}

func (p *UnleashProvider) Resources(ctx context.Context) []func() resource.Resource {
	// return []func() resource.Resource{
	// 	NewExampleResource,
	// }
	return nil
}

func (p *UnleashProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &UnleashProvider{
			version: version,
		}
	}
}
