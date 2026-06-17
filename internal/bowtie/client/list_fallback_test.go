package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Some Controller versions serve a nested collection's index route without a
// trailing slash and 404 the slashed form. These tests pin the client's
// tolerance for both shapes.
func TestGetDeviceGroupsFallsBackWhenSlashed404s(t *testing.T) {
	var hitSlashed, hitBare bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/-net/api/v0/device_group/":
			hitSlashed = true
			w.WriteHeader(http.StatusNotFound)
		case "/-net/api/v0/device_group":
			hitBare = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"a":{"id":"a","name":"A"}}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	groups, err := newTestClient(t, ts).GetDeviceGroups()
	if err != nil {
		t.Fatalf("GetDeviceGroups: %v", err)
	}
	if !hitSlashed || !hitBare {
		t.Errorf("expected both forms to be tried (slashed=%v, bare=%v)", hitSlashed, hitBare)
	}
	if g, ok := groups["a"]; !ok || g.Name != "A" {
		t.Errorf("unexpected groups: %+v", groups)
	}
}

func TestGetCollectionsUsesSlashedFormWhenItWorks(t *testing.T) {
	var paths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer ts.Close()

	if _, err := newTestClient(t, ts).GetCollections(); err != nil {
		t.Fatalf("GetCollections: %v", err)
	}
	// The slashed form works, so the bare form must never be requested.
	if len(paths) != 1 || paths[0] != "/-net/api/v0/collection/" {
		t.Errorf("expected only the slashed form to be tried, got %v", paths)
	}
}

func TestGetListJSONPropagatesNon404Errors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	if _, err := newTestClient(t, ts).GetDeviceGroups(); err == nil {
		t.Fatal("expected a 500 to propagate as an error")
	}
}
