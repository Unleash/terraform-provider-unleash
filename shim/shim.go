// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package shim exposes the provider constructor to external Go modules.
//
// The provider implementation lives in internal/provider, which Go's module
// visibility rules make unimportable from outside this repository. Re-exporting
// New here lets external tooling embed the provider directly — in particular a
// Pulumi provider built on the Pulumi Terraform Bridge, which must construct the
// terraform-plugin-framework provider to generate its schema and language SDKs.
package shim

import (
	"github.com/hashicorp/terraform-plugin-framework/provider"

	internalprovider "github.com/Unleash/terraform-provider-unleash/internal/provider"
)

// New returns the Unleash provider constructor, identical to the one used by
// the provider's own main entrypoint.
func New(version string) func() provider.Provider {
	return internalprovider.New(version)
}
