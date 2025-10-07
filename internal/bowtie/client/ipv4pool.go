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
	var payload OrgIpv4Range = OrgIpv4Range{
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
	fmt.Printf("[DEBUG] UpsertIPv4Pool POST to: %s\n", url)
	fmt.Printf("[DEBUG] UpsertIPv4Pool payload: %s\n", string(body))

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	responseBody, err := c.doRequest(req)
	fmt.Printf("[DEBUG] UpsertIPv4Pool response body: %s\n", string(responseBody))
	fmt.Printf("[DEBUG] UpsertIPv4Pool error: %v\n", err)
	
	if err != nil {
		return fmt.Errorf("failed to upsert IPv4 pool: %w", err)
	}
	
	return nil
}

// DeleteIPv4Pool deletes an IPv4 pool by ID
func (c *Client) DeleteIPv4Pool(id string) error {
	url := c.getHostURL(fmt.Sprintf("/organization/ipv4/%s", id))
	fmt.Printf("[DEBUG] DeleteIPv4Pool DELETE to: %s\n", url)
	
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	responseBody, err := c.doRequest(req)
	fmt.Printf("[DEBUG] DeleteIPv4Pool response: %s\n", string(responseBody))
	fmt.Printf("[DEBUG] DeleteIPv4Pool error: %v\n", err)
	
	return err
}

// GetIPv4Pools retrieves all IPv4 pools for the organization
func (c *Client) GetIPv4Pools() (map[string]OrgIpv4Range, error) {
	url := c.getHostURL("/organization/ipv4")
	fmt.Printf("[DEBUG] GetIPv4Pools GET from: %s\n", url)
	
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	responseBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG] GetIPv4Pools raw response: '%s'\n", string(responseBody))
	fmt.Printf("[DEBUG] GetIPv4Pools response length: %d\n", len(responseBody))

	// Handle empty response or empty object
	if len(responseBody) == 0 {
		fmt.Printf("[DEBUG] Empty response body, returning empty map\n")
		return map[string]OrgIpv4Range{}, nil
	}

	// Trim whitespace and check if it's just an empty object
	trimmed := bytes.TrimSpace(responseBody)
	if len(trimmed) == 0 || string(trimmed) == "{}" {
		fmt.Printf("[DEBUG] Empty object response, returning empty map\n")
		return map[string]OrgIpv4Range{}, nil
	}

	var ipv4Pools map[string]OrgIpv4Range
	err = json.Unmarshal(responseBody, &ipv4Pools)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal IPv4 pools: %w (body: %s)", err, string(responseBody))
	}
	
	fmt.Printf("[DEBUG] Successfully parsed %d IPv4 pools\n", len(ipv4Pools))
	return ipv4Pools, nil
}

// GetIPv4PoolByID retrieves a specific IPv4 pool by ID
func (c *Client) GetIPv4PoolByID(id string) (*OrgIpv4Range, error) {
	url := c.getHostURL(fmt.Sprintf("/organization/ipv4/%s", id))
	fmt.Printf("[DEBUG] GetIPv4PoolByID GET from: %s\n", url)
	
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	responseBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG] GetIPv4PoolByID response: %s\n", string(responseBody))

	var pool OrgIpv4Range
	err = json.Unmarshal(responseBody, &pool)
	return &pool, err
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
	fmt.Printf("[DEBUG] AssignIPv4ToDevice POST to: %s\n", url)
	fmt.Printf("[DEBUG] AssignIPv4ToDevice payload: %s\n", string(body))

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	responseBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[DEBUG] AssignIPv4ToDevice response: %s\n", string(responseBody))

	var response AssignIpv4ToDeviceResponse
	err = json.Unmarshal(responseBody, &response)
	return &response, err
}
