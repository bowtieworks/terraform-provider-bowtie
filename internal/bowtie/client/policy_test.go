package client

import (
	"encoding/json"
	"reflect"
	"testing"
)

// jsonEqual compares two JSON documents structurally, ignoring key order and
// insignificant whitespace.
func jsonEqual(t *testing.T, got []byte, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("got is not valid JSON: %v (%s)", err, got)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("want is not valid JSON: %v (%s)", err, want)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("JSON mismatch\n got: %s\nwant: %s", got, want)
	}
}

// The wire shapes below are taken verbatim from the Controller's SourcePredicate
// serde enum and the frontend policy editor, so these tests pin the client to
// the exact format the policy engine expects.
func TestPredicateMarshal(t *testing.T) {
	cases := []struct {
		name      string
		predicate BowtiePredicate
		want      string
	}{
		{"always", BowtiePredicate{Always: true}, `"Always"`},
		{"authenticated_user", BowtiePredicate{AuthenticatedUser: true}, `"AuthenticatedUser"`},
		{"user", BowtiePredicate{User: "11111111-1111-1111-1111-111111111111"}, `{"User":"11111111-1111-1111-1111-111111111111"}`},
		{"device", BowtiePredicate{Device: "22222222-2222-2222-2222-222222222222"}, `{"Device":"22222222-2222-2222-2222-222222222222"}`},
		{"in_user_group", BowtiePredicate{InUserGroup: "5b5d3ec8-538f-461f-a444-e9f4d4010b8b"}, `{"InUserGroup":"5b5d3ec8-538f-461f-a444-e9f4d4010b8b"}`},
		{"in_device_group", BowtiePredicate{InDeviceGroup: "44444444-4444-4444-4444-444444444444"}, `{"InDeviceGroup":"44444444-4444-4444-4444-444444444444"}`},
		{
			"and",
			BowtiePredicate{And: []BowtiePolicySource{
				{ID: "aaaa", Predicate: BowtiePredicate{InUserGroup: "grp"}},
				{ID: "bbbb", Predicate: BowtiePredicate{InDeviceGroup: "dg"}},
			}},
			`{"And":[{"id":"aaaa","predicate":{"InUserGroup":"grp"}},{"id":"bbbb","predicate":{"InDeviceGroup":"dg"}}]}`,
		},
		{
			"or",
			BowtiePredicate{Or: []BowtiePolicySource{{ID: "cccc", Predicate: BowtiePredicate{AuthenticatedUser: true}}}},
			`{"Or":[{"id":"cccc","predicate":"AuthenticatedUser"}]}`,
		},
		{
			"nor",
			BowtiePredicate{Nor: []BowtiePolicySource{{ID: "dddd", Predicate: BowtiePredicate{Always: true}}}},
			`{"Nor":[{"id":"dddd","predicate":"Always"}]}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.predicate)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			jsonEqual(t, got, tc.want)
		})
	}
}

func TestPredicateMarshalEmptyIsError(t *testing.T) {
	if _, err := json.Marshal(BowtiePredicate{}); err == nil {
		t.Fatal("expected an error marshaling an empty predicate")
	}
}

func TestPredicateUnmarshal(t *testing.T) {
	cases := []struct {
		name string
		json string
		want BowtiePredicate
	}{
		{"always", `"Always"`, BowtiePredicate{Always: true}},
		{"authenticated_user", `"AuthenticatedUser"`, BowtiePredicate{AuthenticatedUser: true}},
		{"user", `{"User":"u-1"}`, BowtiePredicate{User: "u-1"}},
		{"device", `{"Device":"d-1"}`, BowtiePredicate{Device: "d-1"}},
		{"in_user_group", `{"InUserGroup":"ug-1"}`, BowtiePredicate{InUserGroup: "ug-1"}},
		{"in_device_group", `{"InDeviceGroup":"dg-1"}`, BowtiePredicate{InDeviceGroup: "dg-1"}},
		{
			"and",
			`{"And":[{"id":"x","predicate":{"User":"u-1"}}]}`,
			BowtiePredicate{And: []BowtiePolicySource{{ID: "x", Predicate: BowtiePredicate{User: "u-1"}}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got BowtiePredicate
			if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("unmarshal mismatch\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestPredicateUnmarshalRejectsUnknown(t *testing.T) {
	for _, raw := range []string{`"Nonsense"`, `{"Bogus":"x"}`, `{"User":"a","Device":"b"}`} {
		var p BowtiePredicate
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Errorf("expected error unmarshaling %s", raw)
		}
	}
}

func TestPredicateRoundTrip(t *testing.T) {
	predicates := []BowtiePredicate{
		{Always: true},
		{AuthenticatedUser: true},
		{User: "u"},
		{Device: "d"},
		{InUserGroup: "ug"},
		{InDeviceGroup: "dg"},
		{And: []BowtiePolicySource{{ID: "1", Predicate: BowtiePredicate{User: "u"}}}},
		{Or: []BowtiePolicySource{{ID: "2", Predicate: BowtiePredicate{Device: "d"}}}},
		{Nor: []BowtiePolicySource{{ID: "3", Predicate: BowtiePredicate{Always: true}}}},
	}

	for _, want := range predicates {
		encoded, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("marshal %+v: %v", want, err)
		}
		var got BowtiePredicate
		if err := json.Unmarshal(encoded, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", encoded, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("round trip mismatch\n got: %+v\nwant: %+v", got, want)
		}
	}
}

// TestPolicyMarshalMatchesController pins the full policy body, including the
// order/status gotchas, to the shape the upsert endpoint accepts.
func TestPolicyMarshalMatchesController(t *testing.T) {
	order := int64(4)
	policy := BowtiePolicy{
		ID: "4357c170-1a51-495e-b172-81ea0b2d1e78",
		Source: BowtiePolicySource{
			ID:        "43b1fe82-da57-46b4-a8a6-e6a219dd4d9b",
			Predicate: BowtiePredicate{InUserGroup: "5b5d3ec8-538f-461f-a444-e9f4d4010b8b"},
		},
		Dest:   "e805e590-dc2c-4ffc-ab65-9946bc5f16b7",
		Action: "Accept",
		Order:  &order,
		Status: "Enabled",
	}

	got, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonEqual(t, got, `{
		"id": "4357c170-1a51-495e-b172-81ea0b2d1e78",
		"source": {
			"id": "43b1fe82-da57-46b4-a8a6-e6a219dd4d9b",
			"predicate": {"InUserGroup": "5b5d3ec8-538f-461f-a444-e9f4d4010b8b"}
		},
		"dest": "e805e590-dc2c-4ffc-ab65-9946bc5f16b7",
		"action": "Accept",
		"order": 4,
		"status": "Enabled"
	}`)
}

func TestPolicyMarshalOmitsUnsetOrder(t *testing.T) {
	policy := BowtiePolicy{
		ID:     "p-1",
		Source: BowtiePolicySource{ID: "s-1", Predicate: BowtiePredicate{Always: true}},
		Dest:   "rg-1",
		Action: "Reject",
	}

	got, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// An omitted order lets the Controller append the policy; an omitted status
	// defaults to Enabled. Neither key should appear.
	jsonEqual(t, got, `{
		"id": "p-1",
		"source": {"id": "s-1", "predicate": "Always"},
		"dest": "rg-1",
		"action": "Reject"
	}`)
}

// TestPolicyUnmarshalFromController parses a response body taken from the
// Controller's GET /policy document.
func TestPolicyUnmarshalFromController(t *testing.T) {
	body := `{
		"id": "7f0a90bc-62d1-40ac-89bf-8a1eb07c2c7f",
		"source": {
			"id": "067ed702-55e9-4d4d-95c1-14c22e917e17",
			"predicate": {"InUserGroup": "5b5d3ec8-538f-461f-a444-e9f4d4010b8b"}
		},
		"dest": "e805e590-dc2c-4ffc-ab65-9946bc5f16b7",
		"action": "Accept",
		"order": 1,
		"status": "Enabled"
	}`

	var policy BowtiePolicy
	if err := json.Unmarshal([]byte(body), &policy); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if policy.Source.Predicate.InUserGroup != "5b5d3ec8-538f-461f-a444-e9f4d4010b8b" {
		t.Errorf("unexpected predicate: %+v", policy.Source.Predicate)
	}
	if policy.Order == nil || *policy.Order != 1 {
		t.Errorf("unexpected order: %v", policy.Order)
	}
	if policy.Action != "Accept" || policy.Status != "Enabled" {
		t.Errorf("unexpected action/status: %s/%s", policy.Action, policy.Status)
	}
}
