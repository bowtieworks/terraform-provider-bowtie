package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOrgRangeJSONCasing(t *testing.T) {
	ipv4, err := json.Marshal(OrgIPv4Range{
		ID:                      "v4",
		Range:                   "192.0.2.0/24",
		AssignAddressesFromHere: "never",
		SkipFirstNAddresses:     1,
		SiteStrategies:          json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("marshal ipv4: %v", err)
	}
	jsonEqual(t, ipv4, `{
		"id":"v4",
		"range":"192.0.2.0/24",
		"assign-addresses-from-here":"never",
		"skip-first-n-addresses":1,
		"site-strategies":{}
	}`)

	ipv6, err := json.Marshal(OrgIPv6Range{
		ID:                      "v6",
		Range:                   "fd00::/64",
		AssignAddressesFromHere: true,
	})
	if err != nil {
		t.Fatalf("marshal ipv6: %v", err)
	}
	jsonEqual(t, ipv6, `{
		"id":"v6",
		"range":"fd00::/64",
		"assign_addresses_from_here":true
	}`)
}

func TestGetIPv4RangeNotFoundReturnsHTTP404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := newTestClient(t, ts).GetIPv4Range("missing")
	if err == nil {
		t.Fatal("expected GetIPv4Range to return an error")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("expected HTTP 404 error, got %q", err.Error())
	}
}

func TestGetIPv6RangeNotFoundIsActionable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[]`)
	}))
	defer ts.Close()

	_, err := newTestClient(t, ts).GetIPv6Range("missing")
	if err == nil {
		t.Fatal("expected GetIPv6Range to return a not-found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found error, got %q", err.Error())
	}
}
