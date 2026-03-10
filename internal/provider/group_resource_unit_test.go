package provider

import (
	"context"
	"testing"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPopulateGroupStateFromAPI_PreservesEmptyValues(t *testing.T) {
	t.Parallel()

	groupID := int32(42)
	group := unleash.NewGroupSchema("example-group")
	group.Id = &groupID
	group.SetDescription("")
	group.SetRootRoleNil()

	var diagnostics diag.Diagnostics
	state := groupResourceModel{
		MappingsSSO: types.ListValueMust(types.StringType, []attr.Value{}),
		Users:       types.ListValueMust(types.Int64Type, []attr.Value{}),
	}

	populateGroupStateFromAPI(context.Background(), group, &state, &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if !state.Description.IsNull() {
		t.Fatalf("expected empty description to normalize to null")
	}
	if !state.RootRole.IsNull() {
		t.Fatalf("expected nil root role to map to null state")
	}
	if state.MappingsSSO.IsNull() {
		t.Fatalf("expected empty mappings list, got null")
	}
	if got := len(state.MappingsSSO.Elements()); got != 0 {
		t.Fatalf("expected empty mappings list, got %d items", got)
	}
	if state.Users.IsNull() {
		t.Fatalf("expected empty users list, got null")
	}
	if got := len(state.Users.Elements()); got != 0 {
		t.Fatalf("expected empty users list, got %d items", got)
	}
}

func TestPopulateGroupStateFromAPI_PreservesNullForUnsetLists(t *testing.T) {
	t.Parallel()

	groupID := int32(7)
	group := unleash.NewGroupSchema("null-lists")
	group.Id = &groupID

	var diagnostics diag.Diagnostics
	state := groupResourceModel{
		MappingsSSO: types.ListNull(types.StringType),
		Users:       types.ListNull(types.Int64Type),
	}

	populateGroupStateFromAPI(context.Background(), group, &state, &diagnostics)

	if diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if !state.MappingsSSO.IsNull() {
		t.Fatalf("expected mappings to remain null when prior state was null")
	}
	if !state.Users.IsNull() {
		t.Fatalf("expected users to remain null when prior state was null")
	}
}

func TestPopulateGroupStateFromAPI_AddsDiagnosticForNilID(t *testing.T) {
	t.Parallel()

	group := unleash.NewGroupSchema("missing-id")
	var diagnostics diag.Diagnostics
	var state groupResourceModel

	populateGroupStateFromAPI(context.Background(), group, &state, &diagnostics)

	if !diagnostics.HasError() {
		t.Fatal("expected diagnostics error for nil group ID")
	}
}
