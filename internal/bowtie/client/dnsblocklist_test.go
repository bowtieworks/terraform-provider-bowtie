package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpsertDNSBlockListSendsIsAllowlistFalse(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("server could not decode DNS block list: %v", err)
		}
	}))
	defer ts.Close()

	c := newTestClient(t, ts)

	err := c.UpsertDNSBlockList("block-list-1", "Block List", "https://example.com/block.txt", "example.com")
	if err != nil {
		t.Fatalf("UpsertDNSBlockList: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/-net/api/v0/dns_block_list" {
		t.Errorf("path = %s, want /-net/api/v0/dns_block_list", gotPath)
	}

	isAllowlist, present := gotBody["is_allowlist"]
	if !present {
		t.Fatal("is_allowlist was not sent")
	}
	if isAllowlist != false {
		t.Errorf("is_allowlist = %v, want false", isAllowlist)
	}
}
