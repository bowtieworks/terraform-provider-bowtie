package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResourceLocationToClientRejectsAmbiguousLocation(t *testing.T) {
	location := &resourceLocationModel{
		IP:         types.StringValue("10.0.0.10"),
		CIDR:       types.StringValue("10.0.0.0/24"),
		DNS:        types.StringNull(),
		Collection: types.StringNull(),
	}

	_, diags := resourceLocationToClient(location, true)
	if !diags.HasError() {
		t.Fatal("expected ambiguous location to produce a diagnostic")
	}
	if !strings.Contains(diags[0].Detail(), "exactly one") {
		t.Fatalf("unexpected diagnostic: %s", diags[0].Detail())
	}
}

func TestResourceLocationToClientRejectsCollectionWhenTaggedLocationsDisabled(t *testing.T) {
	location := &resourceLocationModel{
		IP:         types.StringNull(),
		CIDR:       types.StringNull(),
		DNS:        types.StringNull(),
		Collection: types.StringValue("collection-id"),
	}

	_, diags := resourceLocationToClient(location, false)
	if !diags.HasError() {
		t.Fatal("expected collection location with legacy format to produce a diagnostic")
	}
	if !strings.Contains(diags[0].Detail(), "tagged_locations") {
		t.Fatalf("unexpected diagnostic: %s", diags[0].Detail())
	}
}

func TestCollectionLocationToAPIRejectsAmbiguousLocation(t *testing.T) {
	location := collectionLocationModel{
		IP:         types.StringValue("10.0.0.10"),
		CIDR:       types.StringNull(),
		DNS:        types.StringValue("internal.example.com"),
		Collection: types.StringNull(),
	}

	_, err := location.toAPI()
	if err == nil {
		t.Fatal("expected ambiguous collection member location to error")
	}
	if !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectionMemberIDsUseMapKeys(t *testing.T) {
	ids := collectionMemberIDs(map[string]client.BowtieCollectionMember{
		"map-key-not-id": {ID: "member-id"},
		"fallback-id":    {},
	})

	if len(ids) != 2 {
		t.Fatalf("expected two IDs, got %v", ids)
	}

	got := map[string]bool{}
	for _, id := range ids {
		got[id] = true
	}
	if !got["fallback-id"] {
		t.Fatalf("expected map key to be used, got %v", ids)
	}
	if !got["map-key-not-id"] {
		t.Fatalf("expected map key to be used even when payload ID is present, got %v", ids)
	}
	if got["member-id"] {
		t.Fatalf("payload ID was used, but the remove endpoint expects map keys: %v", ids)
	}
}

func TestStringSetToMapRejectsUnsupportedField(t *testing.T) {
	ctx := context.Background()
	value, setDiags := types.SetValueFrom(ctx, types.StringType, []string{"not_supported"})
	if setDiags.HasError() {
		t.Fatalf("building set: %v", setDiags)
	}

	var diags diag.Diagnostics
	allowed := map[string]struct{}{"supported": {}}
	stringSetToMap(ctx, value, path.Root("clear_fields"), allowed, &diags)

	if !diags.HasError() {
		t.Fatal("expected unsupported field to produce a diagnostic")
	}
	if !strings.Contains(diags[0].Detail(), "not_supported") {
		t.Fatalf("unexpected diagnostic: %s", diags[0].Detail())
	}
}

func TestTaggedValueFromPlanOmitsValueForNoValueVariant(t *testing.T) {
	tagged := taggedValueFromPlan(
		types.StringValue("manual"),
		types.StringValue("24.03.001"),
		controllerVersionStrategyValueVariants,
	)

	if tagged.Type != "manual" {
		t.Fatalf("Type = %q, want manual", tagged.Type)
	}
	if tagged.Value != nil {
		t.Fatalf("Value = %q, want nil for a no-value variant", *tagged.Value)
	}
}

func TestTaggedValueFromPlanKeepsValueForValueVariant(t *testing.T) {
	tagged := taggedValueFromPlan(
		types.StringValue("specific"),
		types.StringValue("24.03.001"),
		controllerVersionStrategyValueVariants,
	)

	if tagged.Value == nil || *tagged.Value != "24.03.001" {
		t.Fatalf("Value = %v, want 24.03.001", tagged.Value)
	}
}

func TestTaggedValueToStateDropsStaleValueForNoValueVariant(t *testing.T) {
	typeAttr, valueAttr := taggedValueToState(
		client.TaggedValue{Type: "no-delay", Value: stringPointer("15min")},
		controllerVersionStrategySplayValueVariants,
	)

	if typeAttr.ValueString() != "no-delay" {
		t.Fatalf("type = %q, want no-delay", typeAttr.ValueString())
	}
	if !valueAttr.IsNull() {
		t.Fatalf("value = %q, want null", valueAttr.ValueString())
	}
}

func TestIsNotFoundError(t *testing.T) {
	for _, msg := range []string{"HTTP 404 Not Found: missing", "ipv6 range not found: id"} {
		if !isNotFoundError(errString(msg)) {
			t.Fatalf("expected %q to be treated as not found", msg)
		}
	}
}

func stringPointer(s string) *string {
	return &s
}

type errString string

func (e errString) Error() string {
	return string(e)
}
