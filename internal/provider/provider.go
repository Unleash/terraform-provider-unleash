// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"strings"

	unleash "github.com/Unleash/unleash-server-api-go/client"

	"github.com/fatih/structs"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

// ScaffoldingProviderMofunc (p *UnleashProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {del describes the provider data model.
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
				Optional:            true,
			},
			"authorization": schema.StringAttribute{
				MarkdownDescription: "Authhorization token for Unleash API",
				Optional:            true,
				Sensitive:           true,
			},
		},
		Description: "Interface with Unleash server API.",
	}
}

func configValue(configValue basetypes.StringValue, env string) string {
	if configValue.IsNull() {
		return os.Getenv(env)
	}
	return configValue.ValueString()
}

func mustHave(name string, value string, resp *provider.ConfigureResponse) {
	if value == "" {
		resp.Diagnostics.AddError(
			"Unable to find "+name,
			name+" cannot be an empty string",
		)
	}
}

func (p *UnleashProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config UnleashConfiguration

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	base_url := configValue(config.BaseUrl, "UNLEASH_URL")
	authorization := configValue(config.Authorization, "AUTH_TOKEN")
	mustHave("base_url", base_url, resp)
	mustHave("authorization", authorization, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }
	tflog.Info(ctx, "Configuring Unleash client", structs.Map(config))
	tflog.Info(ctx, "Base URL: "+base_url)
	unleashConfig := unleash.NewConfiguration()
	unleashConfig.Servers = unleash.ServerConfigurations{
		unleash.ServerConfiguration{
			URL:         base_url,
			Description: "Unleash server",
		},
	}
	unleashConfig.AddDefaultHeader("Authorization", authorization)

	logLevel := strings.ToLower(os.Getenv("TF_LOG"))
	isDebug := logLevel == "debug" || logLevel == "trace"
	unleashConfig.HTTPClient = httpClient(isDebug)
	client := unleash.NewAPIClient(unleashConfig)

	// Make the Inventory client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
	tflog.Info(ctx, "Configured Unleash client", map[string]any{"success": true})
}

func (p *UnleashProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewProjectResource,
	}
}

func (p *UnleashProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewProjectDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &UnleashProvider{}
	}
}
