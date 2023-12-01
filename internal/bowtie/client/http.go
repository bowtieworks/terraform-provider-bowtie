package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

type Client struct {
	HTTPClient *http.Client

	hostURL string
	auth    AuthPayload
}

type AuthPayload struct {
	Username string `json:"email"`
	Password string `json:"password"`
}

const apiVersionPrefix = "/-net/api/v0"

func NewClient(ctx context.Context, host, username, password string, lazy_auth bool) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c := &Client{
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		hostURL: host,
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

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	// Pre-flight check to ensure that login cookies are present:
	if len(c.HTTPClient.Jar.Cookies(req.URL)) == 0 {
		// Without any cookies for this URL, login first:
		if err := c.Login(); err != nil {
			return nil, err
		}
	}

	if req.Method == http.MethodPost {
		req.Header.Add("Content-Type", "application/json")
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *Client) getHostURL(path string) string {
	if !strings.HasPrefix(path, "/") {
		return ""
	}
	return fmt.Sprintf("%s%s%s", c.hostURL, apiVersionPrefix, path)
}
