package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"sync"
	"time"
)

type Client struct {
	HTTPClient       *http.Client
	Tagged_locations bool

	hostURL   string
	auth      AuthPayload
	authCheck sync.Mutex
}

type AuthPayload struct {
	Username string `json:"email"`
	Password string `json:"password"`
}

const apiVersionPrefix = "/-net/api/v0"

func NewClient(host, username, password string, lazy_auth, tagged_locations, insecure bool, caBundle string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	transport, err := buildTransport(insecure, caBundle)
	if err != nil {
		return nil, err
	}

	c := &Client{
		HTTPClient: &http.Client{
			Timeout:   10 * time.Second,
			Jar:       jar,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Tagged_locations: tagged_locations,
		hostURL:          host,
		auth: AuthPayload{
			Username: username,
			Password: password,
		},
	}

	if !lazy_auth {
		if err := c.Login(); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// buildTransport clones the default transport and applies the provider's TLS
// settings: skipping verification entirely (insecure) or trusting an additional
// CA bundle (PEM contents or a path to a PEM file) for private-CA controllers.
func buildTransport(insecure bool, caBundle string) (*http.Transport, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	tlsConfig := &tls.Config{InsecureSkipVerify: insecure}

	if caBundle != "" {
		pem, err := caBundlePEM(caBundle)
		if err != nil {
			return nil, fmt.Errorf("failed to read ca_bundle: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("ca_bundle did not contain any valid PEM certificates")
		}
		tlsConfig.RootCAs = pool
	}

	transport.TLSClientConfig = tlsConfig
	return transport, nil
}

// caBundlePEM resolves the ca_bundle attribute, which may be inline PEM or a
// path to a PEM file on disk.
func caBundlePEM(caBundle string) ([]byte, error) {
	if strings.Contains(caBundle, "-----BEGIN") {
		return []byte(caBundle), nil
	}
	return os.ReadFile(caBundle)
}

// Check that the client has a login cookie, and if not, authenticate
func (c *Client) ensureAuth(req *http.Request) error {
	// Wrapped in a mutex lock to ensure that we don’t spam auth
	// requests in the event of parallel resources being checked.
	c.authCheck.Lock()
	defer c.authCheck.Unlock()

	if len(c.HTTPClient.Jar.Cookies(req.URL)) == 0 {
		// Without any cookies for this URL, login first:
		if err := c.Login(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	// Pre-flight check to ensure that login cookies are present.
	if err := c.ensureAuth(req); err != nil {
		return nil, err
	}

	if req.Method == http.MethodPost {
		req.Header.Add("Content-Type", "application/json")
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 400 {
		ct := res.Header.Get("Content-Type")
		snippet := string(body)
		if len(snippet) > 4096 {
			snippet = snippet[:4096] + "…[truncated]"
		}
		return nil, fmt.Errorf("HTTP %d %s (%s): %s",
			res.StatusCode, http.StatusText(res.StatusCode), ct, strings.TrimSpace(snippet))
	}

	return body, nil
}

func (c *Client) getHostURL(path string) string {
	if !strings.HasPrefix(path, "/") {
		return ""
	}
	return fmt.Sprintf("%s%s%s", c.hostURL, apiVersionPrefix, path)
}

// doJSONWithFallback sends the request to the first of the supplied paths that
// does not return a 404, unmarshaling the response into out (when non-nil).
// Controller versions disagree on whether a nested index route carries a
// trailing slash, so callers pass both forms and the client uses whichever the
// Controller serves.
func (c *Client) doJSONWithFallback(method string, body []byte, out any, paths ...string) error {
	var lastErr error
	for _, path := range paths {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}

		req, err := http.NewRequest(method, c.getHostURL(path), reader)
		if err != nil {
			return err
		}

		respBody, err := c.doRequest(req)
		if err != nil {
			lastErr = err
			if strings.Contains(err.Error(), "HTTP 404") {
				continue
			}
			return err
		}

		if out != nil {
			return json.Unmarshal(respBody, out)
		}
		return nil
	}
	return lastErr
}

// getListJSON GETs the first of the supplied paths that does not 404.
func (c *Client) getListJSON(out any, paths ...string) error {
	return c.doJSONWithFallback(http.MethodGet, nil, out, paths...)
}
