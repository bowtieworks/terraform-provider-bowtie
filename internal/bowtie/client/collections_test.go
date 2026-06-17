package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRemoveCollectionMembersSendsOneRequestPerMember(t *testing.T) {
	var gotMembers []string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/-net/api/v0/collection/removemember" {
			t.Errorf("path = %s, want /-net/api/v0/collection/removemember", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var got removeCollectionMembers
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("server could not decode removal: %v", err)
		}
		if got.CollectionID != "collection-1" {
			t.Errorf("collection_id = %s, want collection-1", got.CollectionID)
		}
		if len(got.Members) != 1 {
			t.Errorf("members = %v, want exactly one per request", got.Members)
			return
		}
		gotMembers = append(gotMembers, got.Members[0])
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"updated":true}`)
	}))
	defer ts.Close()

	if err := newTestClient(t, ts).RemoveCollectionMembers("collection-1", []string{"member-1", "member-2", "member-3"}); err != nil {
		t.Fatalf("RemoveCollectionMembers: %v", err)
	}
	if want := []string{"member-1", "member-2", "member-3"}; !reflect.DeepEqual(gotMembers, want) {
		t.Fatalf("removed members = %v, want %v", gotMembers, want)
	}
}
