package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OrgIPv4Range is an organization IPv4 address pool. site_strategies maps a
// site ID to its routing strategy; it is kept as raw JSON so Terraform can
// round-trip strategies it does not yet manage rather than clearing them on
// upsert.
type OrgIPv4Range struct {
	ID                      string          `json:"id,omitempty"`
	Range                   string          `json:"range"`
	AssignAddressesFromHere string          `json:"assign-addresses-from-here"`
	SkipFirstNAddresses     int64           `json:"skip-first-n-addresses"`
	SiteStrategies          json.RawMessage `json:"site-strategies,omitempty"`
}

// OrgIPv6Range is an organization IPv6 address pool. When range is empty the
// Controller generates a Bowtie ULA prefix.
type OrgIPv6Range struct {
	ID                      string `json:"id,omitempty"`
	Range                   string `json:"range,omitempty"`
	AssignAddressesFromHere bool   `json:"assign_addresses_from_here"`
}

func (c *Client) UpsertIPv4Range(r *OrgIPv4Range) (*OrgIPv4Range, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/organization/ipv4"), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	respBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	var out OrgIPv4Range
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetIPv4Range(id string) (*OrgIPv4Range, error) {
	var out OrgIPv4Range
	if err := c.getListJSON(&out, fmt.Sprintf("/organization/ipv4/%s", id)); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteIPv4Range(id string) error {
	// cascade=true so destroy succeeds even when devices hold allocations from
	// the range; without it the Controller rejects the delete with a 400.
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/organization/ipv4/%s?cascade=true", id)), nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req)
	return err
}

func (c *Client) UpsertIPv6Range(r *OrgIPv6Range) (*OrgIPv6Range, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/organization/ipv6"), strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	respBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	var out OrgIPv6Range
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListIPv6Ranges returns every organization IPv6 pool. The Controller serves
// IPv6 reads as a list, so callers filter by ID.
func (c *Client) ListIPv6Ranges() ([]OrgIPv6Range, error) {
	var out []OrgIPv6Range
	if err := c.getListJSON(&out, "/organization/ipv6", "/organization/ipv6/"); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetIPv6Range(id string) (*OrgIPv6Range, error) {
	ranges, err := c.ListIPv6Ranges()
	if err != nil {
		return nil, err
	}
	for i := range ranges {
		if ranges[i].ID == id {
			return &ranges[i], nil
		}
	}
	return nil, fmt.Errorf("ipv6 range not found: %s", id)
}

func (c *Client) DeleteIPv6Range(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/organization/ipv6/%s?cascade=true", id)), nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req)
	return err
}
