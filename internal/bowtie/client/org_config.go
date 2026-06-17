package client

import (
	"encoding/json"
	"net/http"
	"strings"
)

// OrgConfig is the organization configuration singleton, kept as a raw field
// map. The POST endpoint replaces the whole document, so callers read the
// current map, overlay only the keys they manage, and post the whole thing
// back. This preserves every field the provider does not surface.
type OrgConfig map[string]json.RawMessage

func (c *Client) GetOrgConfig() (OrgConfig, error) {
	req, err := http.NewRequest(http.MethodGet, c.getHostURL("/organization/config"), nil)
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	cfg := OrgConfig{}
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Client) UpdateOrgConfig(cfg OrgConfig) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/organization/config"), strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	_, err = c.doRequest(req)
	return err
}
