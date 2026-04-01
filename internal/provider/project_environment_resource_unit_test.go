package provider

import (
	"testing"

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
