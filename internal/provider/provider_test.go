// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	unleashConfig = `terraform {
		required_providers {
			unleash = {
				source = "Unleash/unleash"
				version = ">= 0.0.1"
			}
		}
	}

	provider "unleash" {
		base_url = "http://localhost:4242"
		authorization = "*:*.unleash-insecure-admin-api-token"
	}
	
	`
)

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

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
	envOrDefault("UNLEASH_URL", "http://localhost:4242")
	envOrDefault("AUTH_TOKEN", "*:*.unleash-insecure-admin-api-token")
}
