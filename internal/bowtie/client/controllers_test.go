package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetControllerNotFoundReturnsHTTP404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := newTestClient(t, ts).GetController("missing")
	if err == nil {
		t.Fatal("expected GetController to return an error")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("expected HTTP 404 error, got %q", err.Error())
	}
}
