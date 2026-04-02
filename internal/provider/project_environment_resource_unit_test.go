package provider

import (
	"testing"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestShouldManageChangeRequests(t *testing.T) {
	tests := []struct {
		name                  string
		changeRequestsEnabled types.Bool
		requiredApprovals     types.Int64
		expected              bool
	}{
		{
			name:                  "null bool with null approvals does not manage change requests",
			changeRequestsEnabled: types.BoolNull(),
			requiredApprovals:     types.Int64Null(),
			expected:              false,
		},
		{
			name:                  "enabled change requests are managed",
			changeRequestsEnabled: types.BoolValue(true),
			requiredApprovals:     types.Int64Null(),
			expected:              true,
		},
		{
			name:                  "explicitly disabling change requests is managed",
			changeRequestsEnabled: types.BoolValue(false),
			requiredApprovals:     types.Int64Null(),
			expected:              true,
		},
		{
			name:                  "required approvals imply change request management",
			changeRequestsEnabled: types.BoolValue(false),
			requiredApprovals:     types.Int64Value(2),
			expected:              true,
		},
		{
			name:                  "unknown approvals do not imply change request management",
			changeRequestsEnabled: types.BoolNull(),
			requiredApprovals:     types.Int64Unknown(),
			expected:              false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := shouldManageChangeRequests(test.changeRequestsEnabled, test.requiredApprovals)
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestResetChangeRequestConfig(t *testing.T) {
	model := projectEnvironmentResourceModel{
		ProjectId:             types.StringValue("project"),
		EnvironmentName:       types.StringValue("development"),
		ChangeRequestsEnabled: types.BoolValue(true),
		RequiredApprovals:     types.Int64Value(3),
	}

	model.resetChangeRequestConfig()

	if model.ProjectId.ValueString() != "project" {
		t.Fatalf("expected project id to be preserved, got %s", model.ProjectId.ValueString())
	}
	if model.EnvironmentName.ValueString() != "development" {
		t.Fatalf("expected environment name to be preserved, got %s", model.EnvironmentName.ValueString())
	}
	if model.ChangeRequestsEnabled.ValueBool() {
		t.Fatal("expected change requests to be disabled")
	}
	if !model.RequiredApprovals.IsNull() {
		t.Fatal("expected required approvals to be null")
	}
}

func TestSyncChangeRequestConfigFromApiPreservesUnmanagedState(t *testing.T) {
	model := projectEnvironmentResourceModel{
		ProjectId:             types.StringValue("project"),
		EnvironmentName:       types.StringValue("development"),
		ChangeRequestsEnabled: types.BoolNull(),
		RequiredApprovals:     types.Int64Null(),
	}
	requiredApprovals := float32(2)

	model.syncChangeRequestConfigFromApi([]unleash.ChangeRequestEnvironmentConfigSchema{
		{
			Environment:          "development",
			ChangeRequestEnabled: true,
			RequiredApprovals:    *unleash.NewNullableFloat32(&requiredApprovals),
		},
	})

	if !model.ChangeRequestsEnabled.IsNull() {
		t.Fatal("expected change requests enabled to remain null for unmanaged state")
	}
	if !model.RequiredApprovals.IsNull() {
		t.Fatal("expected required approvals to remain null for unmanaged state")
	}
}

func TestSyncChangeRequestConfigNotFoundPreservesUnmanagedState(t *testing.T) {
	model := projectEnvironmentResourceModel{
		ProjectId:             types.StringValue("project"),
		EnvironmentName:       types.StringValue("development"),
		ChangeRequestsEnabled: types.BoolNull(),
		RequiredApprovals:     types.Int64Null(),
	}

	model.syncChangeRequestConfigNotFound()

	if !model.ChangeRequestsEnabled.IsNull() {
		t.Fatal("expected change requests enabled to remain null for unmanaged state")
	}
	if !model.RequiredApprovals.IsNull() {
		t.Fatal("expected required approvals to remain null for unmanaged state")
	}
}

func TestNormalizeUnmanagedChangeRequestConfig(t *testing.T) {
	model := projectEnvironmentResourceModel{
		ProjectId:             types.StringValue("project"),
		EnvironmentName:       types.StringValue("development"),
		ChangeRequestsEnabled: types.BoolUnknown(),
		RequiredApprovals:     types.Int64Unknown(),
	}

	model.normalizeUnmanagedChangeRequestConfig()

	if !model.ChangeRequestsEnabled.IsNull() {
		t.Fatal("expected change requests enabled to normalize to null for unmanaged state")
	}
	if !model.RequiredApprovals.IsNull() {
		t.Fatal("expected required approvals to normalize to null for unmanaged state")
	}
}

func TestNormalizeUnmanagedChangeRequestConfigPreservesManagedState(t *testing.T) {
	model := projectEnvironmentResourceModel{
		ProjectId:             types.StringValue("project"),
		EnvironmentName:       types.StringValue("development"),
		ChangeRequestsEnabled: types.BoolValue(false),
		RequiredApprovals:     types.Int64Null(),
	}

	model.normalizeUnmanagedChangeRequestConfig()

	if model.ChangeRequestsEnabled.IsNull() {
		t.Fatal("expected managed change requests enabled to be preserved")
	}
	if model.ChangeRequestsEnabled.ValueBool() {
		t.Fatal("expected explicit false value to be preserved")
	}
}
