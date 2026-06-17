package client

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
)

func getThrough(t *testing.T, transport *http.Transport, url string) error {
	t.Helper()
	c := &http.Client{Transport: transport}
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func TestBuildTransportTLSModes(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Default: full verification, self-signed server cert is not trusted.
	verifying, err := buildTransport(false, "")
	if err != nil {
		t.Fatalf("buildTransport: %v", err)
	}
	if err := getThrough(t, verifying, ts.URL); err == nil {
		t.Error("expected a TLS verification error against the self-signed server")
	}

	// insecure: verification skipped, request succeeds.
	insecure, err := buildTransport(true, "")
	if err != nil {
		t.Fatalf("buildTransport: %v", err)
	}
	if insecure.TLSClientConfig == nil || !insecure.TLSClientConfig.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be set")
	}
	if err := getThrough(t, insecure, ts.URL); err != nil {
		t.Errorf("insecure request should succeed: %v", err)
	}

	// ca_bundle: the server's own cert is trusted while still verifying.
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ts.Certificate().Raw})
	trusted, err := buildTransport(false, string(certPEM))
	if err != nil {
		t.Fatalf("buildTransport with ca_bundle: %v", err)
	}
	if trusted.TLSClientConfig.InsecureSkipVerify {
		t.Error("ca_bundle must not disable verification")
	}
	if trusted.TLSClientConfig.RootCAs == nil {
		t.Error("expected RootCAs to be populated from ca_bundle")
	}
	if err := getThrough(t, trusted, ts.URL); err != nil {
		t.Errorf("request with trusted ca_bundle should succeed: %v", err)
	}
}

func TestBuildTransportRejectsInvalidCABundle(t *testing.T) {
	if _, err := buildTransport(false, "-----BEGIN CERTIFICATE-----\nnot base64\n-----END CERTIFICATE-----"); err == nil {
		t.Error("expected an error for a ca_bundle with no valid certificates")
	}
}
