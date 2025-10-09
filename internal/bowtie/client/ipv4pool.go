package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SiteStrategy represents a strategy for a specific site
type SiteStrategy struct {
	Type  string  `json:"type"`
	Value *string `json:"value,omitempty"`
}

// OrgIpv4Range represents an IPv4 range in the organization
type OrgIpv4Range struct {
	ID                      string                  `json:"id"`
	Range                   string                  `json:"range"`
	AssignAddressesFromHere string                  `json:"assign-addresses-from-here"`
	SkipFirstNAddresses     int                     `json:"skip-first-n-addresses"`
	SiteStrategies          map[string]SiteStrategy `json:"site-strategies,omitempty"`
}

// UpsertIPv4Pool creates or updates an IPv4 pool
func (c *Client) UpsertIPv4Pool(id string, ipRange string, assignAddressesFromHere string, skipFirstNAddresses int, siteStrategies map[string]SiteStrategy) error {
	payload := OrgIpv4Range{
		ID:                      id,
		Range:                   ipRange,
		AssignAddressesFromHere: assignAddressesFromHere,
		SkipFirstNAddresses:     skipFirstNAddresses,
		SiteStrategies:          siteStrategies,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := c.getHostURL("/organization/ipv4")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	if err != nil {
		return fmt.Errorf("failed to upsert IPv4 pool: %w", err)
	}
	return nil
}

// DeleteIPv4Pool deletes an IPv4 pool by ID
func (c *Client) DeleteIPv4Pool(id string) error {
	url := c.getHostURL(fmt.Sprintf("/organization/ipv4/%s", id))
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req)
	return err
}

// GetIPv4Pools retrieves all IPv4 pools for the organization
func (c *Client) GetIPv4Pools() (map[string]OrgIpv4Range, error) {
	url := c.getHostURL("/organization/ipv4")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	responseBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	// Handle empty response or empty object
	if len(responseBody) == 0 {
		return map[string]OrgIpv4Range{}, nil
	}

	trimmed := bytes.TrimSpace(responseBody)
	if len(trimmed) == 0 || string(trimmed) == "{}" {
		return map[string]OrgIpv4Range{}, nil
	}

	var ipv4Pools map[string]OrgIpv4Range
	if err := json.Unmarshal(responseBody, &ipv4Pools); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IPv4 pools: %w (body: %s)", err, string(responseBody))
	}
	return ipv4Pools, nil
}

// GetIPv4PoolByID retrieves a specific IPv4 pool by ID
func (c *Client) GetIPv4PoolByID(id string) (*OrgIpv4Range, error) {
	url := c.getHostURL(fmt.Sprintf("/organization/ipv4/%s", id))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	responseBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var pool OrgIpv4Range
	if err := json.Unmarshal(responseBody, &pool); err != nil {
		return nil, err
	}
	return &pool, nil
}

// AssignIpv4ToDeviceRequest represents the request for assigning an IPv4 to a device
type AssignIpv4ToDeviceRequest struct {
	RangeID  string  `json:"range_id"`
	DeviceID string  `json:"device_id"`
	Address  *string `json:"address,omitempty"` // Optional - if not provided, random address assigned
}

// AssignIpv4ToDeviceResponse represents the response from assigning an IPv4 to a device
type AssignIpv4ToDeviceResponse struct {
	RangeID  string `json:"range_id"`
	DeviceID string `json:"device_id"`
	Address  string `json:"address"`
}

// AssignIPv4ToDevice assigns an IPv4 address from a pool to a device
// If address is nil, a random address from the range will be assigned
func (c *Client) AssignIPv4ToDevice(rangeID string, deviceID string, address *string) (*AssignIpv4ToDeviceResponse, error) {
	payload := AssignIpv4ToDeviceRequest{
		RangeID:  rangeID,
		DeviceID: deviceID,
		Address:  address,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := c.getHostURL("/organization/ipv4/assign_to_device")
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	responseBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var response AssignIpv4ToDeviceResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
