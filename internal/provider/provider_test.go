// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/assert"
)

var CheckIsSupportedVersion = checkIsSupportedVersion

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"unleash": providerserver.NewProtocol6WithError(New("test")()),
}

func envOrDefault(name string, defaultValue string) {
	if os.Getenv(name) == "" {
		os.Setenv(name, defaultValue)
	}
}

func testAccPreCheck(_ *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
	envOrDefault("UNLEASH_URL", "http://localhost:4242")
	envOrDefault("AUTH_TOKEN", "*:*.unleash-insecure-admin-api-token")
	envOrDefault("UNLEASH_ENTERPRISE", "false")
}

func currentUnleashPlan() string {
	plan := strings.ToLower(os.Getenv("UNLEASH_PLAN"))
	if plan != "" {
		return plan
	}

	if os.Getenv("UNLEASH_ENTERPRISE") == "true" {
		return "enterprise"
	}

	return "oss"
}

func supportsEnterpriseAcceptanceTests() bool {
	if os.Getenv("UNLEASH_ENTERPRISE") != "true" {
		return false
	}

	switch currentUnleashPlan() {
	case "enterprise", "pro":
		return true
	default:
		return false
	}
}

func skipUnlessEnterpriseCompatiblePlan(t *testing.T) {
	t.Helper()

	if supportsEnterpriseAcceptanceTests() {
		return
	}

	t.Skipf(
		"Skipping enterprise-compatible tests (requires UNLEASH_ENTERPRISE=true and UNLEASH_PLAN=enterprise or pro, got UNLEASH_PLAN=%q)",
		currentUnleashPlan(),
	)
}

func Test_provider_checkIsSupportedVersion_556(t *testing.T) {
	var diags diag.Diagnostics
	CheckIsSupportedVersion("5.5.6", &diags)
	t.Log(diags)
	assert.True(t, diags.HasError())
}

func Test_provider_checkIsSupportedVersion_556_terraform(t *testing.T) {
	var diags diag.Diagnostics
	CheckIsSupportedVersion("5.6.0-terraform-rc", &diags)
	t.Log(diags)
	assert.False(t, diags.HasError())
}

func Test_provider_checkIsSupportedVersion_560(t *testing.T) {
	var diags diag.Diagnostics
	CheckIsSupportedVersion("5.6.0", &diags)
	assert.False(t, diags.HasError())
}

func Test_provider_configValue(t *testing.T) {
	t.Setenv("FOO", "foo")
	t.Setenv("BAR", "bar")
	t.Setenv("BAZ", "baz")
	assert.Equal(t, "foo", configValue(types.StringValue("foo"), "BAR"))
	assert.Equal(t, "", configValue(types.StringNull(), "SOME_ENV"))
	assert.Equal(t, "bar", configValue(types.StringNull(), "BAR"))
	assert.Equal(t, "bar", configValue(types.StringNull(), "BAR", "BAZ"))
	assert.Equal(t, "baz", configValue(types.StringNull(), "QUX", "BAZ"))
	assert.Equal(t, "bar", configValue(types.StringNull(), "QUX", "BAR", "BAZ"))
}

func Test_unleashClient_setsUnleashHeaders(t *testing.T) {
	ctx := context.Background()
	p := &UnleashProvider{version: "1.2.3"}
	cfg := &UnleashConfiguration{
		BaseUrl:       types.StringValue("http://example.com"),
		Authorization: types.StringValue("secret"),
	}

	var diags diag.Diagnostics
	client := unleashClient(ctx, p, cfg, &diags)

	assert.False(t, diags.HasError())
	if assert.NotNil(t, client) {
		cfg := client.GetConfig()
		headers := cfg.DefaultHeader

		assert.Equal(t, "secret", headers["Authorization"])
		assert.Equal(t, terraformProviderAppName(), headers[unleashAppNameHeader])
		assert.NotContains(t, headers, "X-Unleash-InstanceId")
		assert.Equal(t, UserAgent+"/"+p.version, cfg.UserAgent)
	}
}

func Test_currentUnleashPlan(t *testing.T) {
	t.Run("defaults to oss", func(t *testing.T) {
		t.Setenv("UNLEASH_ENTERPRISE", "false")
		t.Setenv("UNLEASH_PLAN", "")
		assert.Equal(t, "oss", currentUnleashPlan())
	})

	t.Run("defaults to enterprise when enterprise env is set", func(t *testing.T) {
		t.Setenv("UNLEASH_ENTERPRISE", "true")
		t.Setenv("UNLEASH_PLAN", "")
		assert.Equal(t, "enterprise", currentUnleashPlan())
	})

	t.Run("uses explicit plan", func(t *testing.T) {
		t.Setenv("UNLEASH_ENTERPRISE", "true")
		t.Setenv("UNLEASH_PLAN", "PrO")
		assert.Equal(t, "pro", currentUnleashPlan())
	})
}

func Test_supportsEnterpriseAcceptanceTests(t *testing.T) {
	t.Run("oss does not support enterprise acceptance tests", func(t *testing.T) {
		t.Setenv("UNLEASH_ENTERPRISE", "false")
		t.Setenv("UNLEASH_PLAN", "")
		assert.False(t, supportsEnterpriseAcceptanceTests())
	})

	t.Run("enterprise plan supports enterprise acceptance tests", func(t *testing.T) {
		t.Setenv("UNLEASH_ENTERPRISE", "true")
		t.Setenv("UNLEASH_PLAN", "enterprise")
		assert.True(t, supportsEnterpriseAcceptanceTests())
	})

	t.Run("pro plan supports enterprise acceptance tests", func(t *testing.T) {
		t.Setenv("UNLEASH_ENTERPRISE", "true")
		t.Setenv("UNLEASH_PLAN", "pro")
		assert.True(t, supportsEnterpriseAcceptanceTests())
	})
}
