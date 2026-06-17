package resources

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// clearSourceIDs recursively blanks the per-operand IDs so that a round-tripped
// predicate can be compared structurally: the model does not track nested
// operand IDs, so toPredicate assigns fresh ones on the way out.
func clearSourceIDs(p *client.BowtiePredicate) {
	for _, group := range [][]client.BowtiePolicySource{p.And, p.Or, p.Nor} {
		for i := range group {
			group[i].ID = ""
			clearSourceIDs(&group[i].Predicate)
		}
	}
}

// nestedPredicate is Or[ And[ InDeviceGroup, AuthenticatedUser ], InDeviceGroup ],
// the two-level shape the Controller produces for composite policies.
func nestedPredicate() client.BowtiePredicate {
	return client.BowtiePredicate{
		Or: []client.BowtiePolicySource{
			{
				ID: "s1",
				Predicate: client.BowtiePredicate{
					And: []client.BowtiePolicySource{
						{ID: "s2", Predicate: client.BowtiePredicate{InDeviceGroup: "dg-1"}},
						{ID: "s3", Predicate: client.BowtiePredicate{AuthenticatedUser: true}},
					},
				},
			},
			{ID: "s4", Predicate: client.BowtiePredicate{InDeviceGroup: "dg-2"}},
		},
	}
}

func TestPredicateToSourceRoundTrip(t *testing.T) {
	want := nestedPredicate()

	model, err := predicateToSource(client.BowtiePolicySource{ID: "top-id", Predicate: want})
	if err != nil {
		t.Fatalf("predicateToSource: unexpected error: %v", err)
	}
	if model.ID.ValueString() != "top-id" {
		t.Errorf("top-level source id: got %q, want %q", model.ID.ValueString(), "top-id")
	}

	// The model must expose the nested shape: Or -> [ {And: [..2..]}, {leaf} ].
	if len(model.Or) != 2 {
		t.Fatalf("model.Or length: got %d, want 2", len(model.Or))
	}
	if len(model.Or[0].And) != 2 {
		t.Fatalf("model.Or[0].And length: got %d, want 2", len(model.Or[0].And))
	}
	if model.Or[0].And[0].DeviceGroup.ValueString() != "dg-1" {
		t.Errorf("nested device_group: got %q, want %q", model.Or[0].And[0].DeviceGroup.ValueString(), "dg-1")
	}
	if !model.Or[0].And[1].AuthenticatedUser.ValueBool() {
		t.Errorf("nested authenticated_user: got false, want true")
	}

	got, err := model.toPredicate()
	if err != nil {
		t.Fatalf("toPredicate: unexpected error: %v", err)
	}

	clearSourceIDs(&got)
	clearSourceIDs(&want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestPredicateToSourceMaxDepth(t *testing.T) {
	// One level deeper than the schema can represent: Or[And[Or[Nor[leaf]]]].
	tooDeep := client.BowtiePredicate{
		Or: []client.BowtiePolicySource{{Predicate: client.BowtiePredicate{
			And: []client.BowtiePolicySource{{Predicate: client.BowtiePredicate{
				Or: []client.BowtiePolicySource{{Predicate: client.BowtiePredicate{
					Nor: []client.BowtiePolicySource{{Predicate: client.BowtiePredicate{Always: true}}},
				}}},
			}}},
		}}},
	}

	_, err := predicateToSource(client.BowtiePolicySource{ID: "top", Predicate: tooDeep})
	if err == nil {
		t.Fatalf("expected an error for over-deep nesting, got nil")
	}
	if !strings.Contains(err.Error(), "maximum supported depth") {
		t.Errorf("error %q does not mention the depth limit", err.Error())
	}
}

func TestPredicateToSourceAtMaxDepth(t *testing.T) {
	// Exactly at the limit: Or[And[Or[leaf]]] must convert without error.
	atLimit := client.BowtiePredicate{
		Or: []client.BowtiePolicySource{{Predicate: client.BowtiePredicate{
			And: []client.BowtiePolicySource{{Predicate: client.BowtiePredicate{
				Or: []client.BowtiePolicySource{{Predicate: client.BowtiePredicate{Always: true}}},
			}}},
		}}},
	}

	if _, err := predicateToSource(client.BowtiePolicySource{ID: "top", Predicate: atLimit}); err != nil {
		t.Fatalf("predicateToSource at max depth: unexpected error: %v", err)
	}
}

func TestSourceToPredicateExactlyOneSelector(t *testing.T) {
	cases := []struct {
		name    string
		source  policySourceModel
		wantErr string
	}{
		{
			name:    "no selector",
			source:  policySourceModel{},
			wantErr: "must set one matcher",
		},
		{
			name: "two selectors",
			source: policySourceModel{
				Always:      types.BoolValue(true),
				DeviceGroup: types.StringValue("dg-1"),
			},
			wantErr: "exactly one matcher",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.source.toPredicate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}
