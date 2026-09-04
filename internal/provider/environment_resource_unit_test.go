package provider

import (
	"testing"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRequiredApprovalsFromApi(t *testing.T) {
	tests := []struct {
		name              string
		requiredApprovals unleash.NullableInt32
		expected          types.Int64
	}{
		{
			name:              "unset approvals map to null",
			requiredApprovals: unleash.NullableInt32{},
			expected:          types.Int64Null(),
		},
		{
			name:              "explicit null approvals map to null",
			requiredApprovals: nullableInt32(nil),
			expected:          types.Int64Null(),
		},
		{
			name:              "zero approvals map to null, since a configuration cannot express them",
			requiredApprovals: nullableInt32(int32Ptr(0)),
			expected:          types.Int64Null(),
		},
		{
			name:              "set approvals map to their value",
			requiredApprovals: nullableInt32(int32Ptr(3)),
			expected:          types.Int64Value(3),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := requiredApprovalsFromApi(test.requiredApprovals)
			if !actual.Equal(test.expected) {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestApplyRequiredApprovalsUpdate(t *testing.T) {
	tests := []struct {
		name          string
		planned       types.Int64
		current       types.Int64
		expectSet     bool
		expectedValue *int32
	}{
		{
			name:      "unmanaged approvals leave the field out of the payload",
			planned:   types.Int64Null(),
			current:   types.Int64Null(),
			expectSet: false,
		},
		{
			name:          "planned approvals are sent",
			planned:       types.Int64Value(2),
			current:       types.Int64Null(),
			expectSet:     true,
			expectedValue: int32Ptr(2),
		},
		{
			name:          "changed approvals are sent",
			planned:       types.Int64Value(5),
			current:       types.Int64Value(2),
			expectSet:     true,
			expectedValue: int32Ptr(5),
		},
		{
			name:          "removing approvals sends an explicit null",
			planned:       types.Int64Null(),
			current:       types.Int64Value(2),
			expectSet:     true,
			expectedValue: nil,
		},
		{
			name:      "unknown approvals with no prior value leave the field out",
			planned:   types.Int64Unknown(),
			current:   types.Int64Null(),
			expectSet: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := *unleash.NewUpdateEnvironmentSchemaWithDefaults()

			applyRequiredApprovalsUpdate(&request, test.planned, test.current)

			if request.RequiredApprovals.IsSet() != test.expectSet {
				t.Fatalf("expected IsSet %t, got %t", test.expectSet, request.RequiredApprovals.IsSet())
			}

			if !test.expectSet {
				return
			}

			actual := request.RequiredApprovals.Get()
			if test.expectedValue == nil {
				if actual != nil {
					t.Fatalf("expected an explicit null, got %d", *actual)
				}
				return
			}

			if actual == nil {
				t.Fatalf("expected %d, got an explicit null", *test.expectedValue)
			}
			if *actual != *test.expectedValue {
				t.Fatalf("expected %d, got %d", *test.expectedValue, *actual)
			}
		})
	}
}

func TestHydrateFromApi(t *testing.T) {
	environment := unleash.NewEnvironmentSchemaWithDefaults()
	environment.Name = "fynbos"
	environment.Type = "semi-arid"
	environment.SetRequiredApprovals(4)

	var model environmentResourceModel
	model.hydrateFromApi(*environment)

	if model.Name.ValueString() != "fynbos" {
		t.Fatalf("expected name fynbos, got %s", model.Name.ValueString())
	}
	if model.Type.ValueString() != "semi-arid" {
		t.Fatalf("expected type semi-arid, got %s", model.Type.ValueString())
	}
	if !model.RequiredApprovals.Equal(types.Int64Value(4)) {
		t.Fatalf("expected 4 required approvals, got %v", model.RequiredApprovals)
	}
}

func int32Ptr(value int32) *int32 {
	return &value
}

func nullableInt32(value *int32) unleash.NullableInt32 {
	nullable := unleash.NullableInt32{}
	nullable.Set(value)
	return nullable
}

func TestRequiredApprovalsWereApplied(t *testing.T) {
	tests := []struct {
		name      string
		requested types.Int64
		applied   types.Int64
		expected  bool
	}{
		{
			name:      "matching values are accepted",
			requested: types.Int64Value(2),
			applied:   types.Int64Value(2),
			expected:  true,
		},
		{
			name:      "an unmanaged attribute is accepted",
			requested: types.Int64Null(),
			applied:   types.Int64Null(),
			expected:  true,
		},
		{
			name:      "a silently ignored request is rejected",
			requested: types.Int64Value(2),
			applied:   types.Int64Null(),
			expected:  false,
		},
		{
			name:      "a server side override is rejected",
			requested: types.Int64Value(2),
			applied:   types.Int64Value(1),
			expected:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var diagnostics diag.Diagnostics

			actual := requiredApprovalsWereApplied(test.requested, test.applied, &diagnostics)

			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}
			if diagnostics.HasError() == test.expected {
				t.Fatalf("expected HasError %t, got %t", !test.expected, diagnostics.HasError())
			}
		})
	}
}
