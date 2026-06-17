package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
)

// newTestClient returns a Client wired to ts with a pre-seeded session cookie so
// that doRequest's just-in-time login is skipped.
func newTestClient(t *testing.T, ts *httptest.Server) *Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: "test"}})

	return &Client{
		HTTPClient: &http.Client{Jar: jar},
		hostURL:    ts.URL,
	}
}

func TestUpsertPolicyRoundTrip(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody BowtiePolicy

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("server could not decode policy: %v", err)
		}

		// The Controller assigns an order when none is supplied.
		if gotBody.Order == nil {
			assigned := int64(7)
			gotBody.Order = &assigned
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gotBody)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)

	saved, err := c.UpsertPolicy(BowtiePolicy{
		ID:     "p-1",
		Source: BowtiePolicySource{ID: "s-1", Predicate: BowtiePredicate{InUserGroup: "ug-1"}},
		Dest:   "rg-1",
		Action: "Accept",
	})
	if err != nil {
		t.Fatalf("UpsertPolicy: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/-net/api/v0/policy/upsert_policy" {
		t.Errorf("path = %s, want /-net/api/v0/policy/upsert_policy", gotPath)
	}
	if gotBody.Source.Predicate.InUserGroup != "ug-1" {
		t.Errorf("server received predicate %+v", gotBody.Source.Predicate)
	}
	if saved.Order == nil || *saved.Order != 7 {
		t.Errorf("expected server-assigned order 7, got %v", saved.Order)
	}
}

func TestGetPolicyParsesDocument(t *testing.T) {
	document := `{
		"policies": {
			"p-1": {
				"id": "p-1",
				"source": {"id": "s-1", "predicate": {"InUserGroup": "ug-1"}},
				"dest": "rg-1",
				"action": "Accept",
				"order": 2,
				"status": "Disabled"
			}
		},
		"resource_groups": {},
		"resources": {}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/-net/api/v0/policy" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, document)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)

	policy, err := c.GetPolicy("p-1")
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if policy.Dest != "rg-1" || policy.Action != "Accept" || policy.Status != "Disabled" {
		t.Errorf("unexpected policy: %+v", policy)
	}
	if policy.Source.Predicate.InUserGroup != "ug-1" {
		t.Errorf("unexpected predicate: %+v", policy.Source.Predicate)
	}
}
