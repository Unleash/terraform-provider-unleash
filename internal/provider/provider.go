// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"

	unleash "github.com/Unleash/unleash-server-api-go/client"

	"github.com/Masterminds/semver"
	"github.com/fatih/structs"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

const (
	UserAgent            = "Terraform-Provider-Unleash"
	unleashAppNameHeader = "X-Unleash-AppName"
	unleashSDKHeader     = "unleash-sdk"
)

// ScaffoldingProviderMofunc (p *UnleashProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {del describes the provider data model.
type UnleashConfiguration struct {
	BaseUrl       types.String `tfsdk:"base_url"`
	Authorization types.String `tfsdk:"authorization"`
}

func (p *UnleashProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "unleash"
	resp.Version = p.version
}

func unleashClient(ctx context.Context, provider *UnleashProvider, config *UnleashConfiguration, diagnostics *diag.Diagnostics) *unleash.APIClient {
	base_url := strings.TrimSuffix(configValue(config.BaseUrl, "UNLEASH_URL"), "/")
	authorization := configValue(config.Authorization, "AUTH_TOKEN", "UNLEASH_AUTH_TOKEN")
	mustHave("base_url", base_url, diagnostics)
	mustHave("authorization", authorization, diagnostics)

	if diagnostics.HasError() {
		return nil
	}

	tflog.Debug(ctx, "Configuring Unleash client", structs.Map(config))
	tflog.Info(ctx, "Base URL: "+base_url)
	unleashConfig := unleash.NewConfiguration()
	unleashConfig.Servers = unleash.ServerConfigurations{
		unleash.ServerConfiguration{
			URL:         base_url,
			Description: "Unleash server",
		},
	}
	unleashConfig.AddDefaultHeader("Authorization", authorization)
	appName := terraformProviderAppName()
	unleashConfig.AddDefaultHeader(unleashAppNameHeader, appName)
	unleashConfig.AddDefaultHeader(unleashSDKHeader, terraformProviderSDKIdentifier(provider.version))
	unleashConfig.UserAgent = fmt.Sprintf("%s/%s", UserAgent, provider.version)

	logLevel := strings.ToLower(os.Getenv("TF_LOG"))
	isDebug := logLevel == "debug" || logLevel == "trace"
	unleashConfig.HTTPClient = httpClient(isDebug)
	client := unleash.NewAPIClient(unleashConfig)

	return client
}

func (p *UnleashProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Unleash base URL (everything before `/api`)",
				Optional:            true,
			},
			"authorization": schema.StringAttribute{
				MarkdownDescription: "Authorization token for Unleash API",
				Optional:            true,
				Sensitive:           true,
			},
		},
		MarkdownDescription: `Interface with [Unleash server API](https://docs.getunleash.io/reference/api/unleash). This provider implements a subset of the operations that can be done with Unleash. The focus is mostly in setting up the instance with projects, roles, permissions, groups, and other typical configuration usually performed by admins.

You can check a complete example [here](https://github.com/Unleash/terraform-provider-unleash/tree/main/examples/staged) under stage_4 folder.`,
	}
}

func configValue(configValue basetypes.StringValue, envs ...string) string {
	if configValue.IsNull() {
		for _, env := range envs {
			if val := os.Getenv(env); val != "" {
				return val
			}
		}
	}
	return configValue.ValueString()
}

func mustHave(name string, value string, diagnostics *diag.Diagnostics) {
	if value == "" {
		diagnostics.AddError(
			"Unable to find "+name,
			name+" cannot be an empty string",
		)
	}
}

func terraformProviderAppName() string {
	return "terraform-provider-unleash"
}

func terraformProviderSDKIdentifier(version string) string {
	return fmt.Sprintf("%s:%s", terraformProviderAppName(), version)
}

func checkIsSupportedVersion(version string, diags *diag.Diagnostics) {
	minimumVersion, _ := semver.NewVersion("5.6.0-0") // -0 is a hack to make 5.6.0 pre release version acceptable
	v, err := semver.NewVersion(version)
	if err != nil {
		diags.AddError(
			fmt.Sprintf("Unable read unleash version from string %s", version),
			err.Error(),
		)
		return
	}

	if v.Compare(minimumVersion) < 0 {
		diags.AddError(
			"Unsupported Unleash version",
			fmt.Sprintf("You're using version %s, while the provider requires at least %s", version, minimumVersion),
		)
		return
	}
}

func (p *UnleashProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config UnleashConfiguration

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	client := unleashClient(ctx, p, &config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

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
		NewApiTokenResource,
		NewRoleResource,
		NewProjectAccessResource,
		NewServiceAccountResource,
		NewServiceAccountTokensResource,
		NewOidcResource,
		NewSamlResource,
		NewContextFieldResource,
		NewEnvironmentResource,
		NewProjectEnvironmentResource,
	}
}

func (p *UnleashProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewProjectDataSource,
		NewPermissionDataSource,
		NewRoleDataSource,
		NewContextFieldDataSource,
		NewEnvironmentDataSource,
		NewProjectEnvironmentDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &UnleashProvider{version}
	}
}
