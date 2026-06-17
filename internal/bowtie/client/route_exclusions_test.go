package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouteExclusionMarshalMatchesController(t *testing.T) {
	pct := 50
	exclusion := BowtieRouteExclusion{
		ID:                    "re-1",
		Name:                  "Office split tunnel",
		CollectionID:          "col-1",
		Sites:                 BowtieSiteDefinition{Type: "specific", Value: []string{"site-1"}},
		ApplyStrategy:         BowtieApplyStrategy{Type: "percentage-user-match", Value: &pct},
		OnlyIfWANMatchesCIDRs: []string{"203.0.113.0/24"},
		MatchOnlyDeviceGroups: []string{"dg-1"},
		MatchOnlyUserGroups:   []string{},
	}

	got, err := json.Marshal(exclusion)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jsonEqual(t, got, `{
		"id": "re-1",
		"name": "Office split tunnel",
		"collection-id": "col-1",
		"sites": {"type": "specific", "value": ["site-1"]},
		"apply-strategy": {"type": "percentage-user-match", "value": 50},
		"only-if-wan-matches-cidrs": ["203.0.113.0/24"],
		"match-only-device-os": null,
		"match-only-device-type": null,
		"match-only-ownership": null,
		"match-only-device-groups": ["dg-1"],
		"match-only-user-groups": []
	}`)
}

func TestSiteDefinitionAndApplyStrategyTags(t *testing.T) {
	all, _ := json.Marshal(BowtieSiteDefinition{Type: "all"})
	jsonEqual(t, all, `{"type":"all"}`)

	always, _ := json.Marshal(BowtieApplyStrategy{Type: "always"})
	jsonEqual(t, always, `{"type":"always"}`)

	specific, _ := json.Marshal(BowtieSiteDefinition{Type: "specific", Value: []string{"s1", "s2"}})
	jsonEqual(t, specific, `{"type":"specific","value":["s1","s2"]}`)
}

func TestRouteExclusionRoundTrip(t *testing.T) {
	pct := 200
	want := BowtieRouteExclusion{
		ID:                    "re-2",
		Name:                  "rollout",
		CollectionID:          "c",
		Sites:                 BowtieSiteDefinition{Type: "all"},
		ApplyStrategy:         BowtieApplyStrategy{Type: "percentage-device-match", Value: &pct},
		OnlyIfWANMatchesCIDRs: []string{},
		MatchOnlyDeviceGroups: []string{},
		MatchOnlyUserGroups:   []string{},
		Version:               "abc123",
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got BowtieRouteExclusion
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Sites.Type != "all" || got.ApplyStrategy.Type != "percentage-device-match" || *got.ApplyStrategy.Value != 200 {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

func TestUpsertRouteExclusionPathAndResponse(t *testing.T) {
	var gotPath, gotMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(body); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer ts.Close()

	saved, err := newTestClient(t, ts).UpsertRouteExclusion(BowtieRouteExclusion{
		ID:            "re-1",
		Name:          "x",
		CollectionID:  "c",
		Sites:         BowtieSiteDefinition{Type: "all"},
		ApplyStrategy: BowtieApplyStrategy{Type: "always"},
	})
	if err != nil {
		t.Fatalf("UpsertRouteExclusion: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/-net/api/v0/route_exclusion/" {
		t.Errorf("unexpected request %s %s", gotMethod, gotPath)
	}
	if saved.CollectionID != "c" {
		t.Errorf("unexpected saved exclusion: %+v", saved)
	}
}
