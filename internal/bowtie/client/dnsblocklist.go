package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) UpsertDNSBlockList(id string, name string, upstream string, override_to_allow string) error {
	var payload DNSBlockList = DNSBlockList{
		ID:              id,
		Name:            name,
		Upstream:        upstream,
		OverrideToAllow: override_to_allow,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/dns_block_list"), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) DeleteDNSBlockList(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/dns_block_list/%s", id)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) GetDNSBlockLists() (map[string]DNSBlockList, error) {
	req, err := http.NewRequest(http.MethodGet, c.getHostURL("/dns_block_list"), nil)
	if err != nil {
		return nil, err
	}

	responseBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var dnsblocklists map[string]DNSBlockList = map[string]DNSBlockList{}
	err = json.Unmarshal(responseBody, &dnsblocklists)
	return dnsblocklists, err
}
