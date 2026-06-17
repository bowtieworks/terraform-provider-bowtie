package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// BowtieDeviceGroup is the device-side analog of a user group. It is referenced
// by policy sources via the InDeviceGroup predicate.
type BowtieDeviceGroup struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// deviceGroupUpsert is the request body for /device_group/upsert. Fields left
// out are preserved server-side; an explicit null clears them. We always send
// name and description so the resource fully owns those fields.
type deviceGroupUpsert struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (c *Client) GetDeviceGroups() (map[string]BowtieDeviceGroup, error) {
	groups := map[string]BowtieDeviceGroup{}
	if err := c.getListJSON(&groups, "/device_group/", "/device_group"); err != nil {
		return nil, err
	}
	return groups, nil
}

func (c *Client) UpsertDeviceGroup(id, name string, description *string) error {
	payload, err := json.Marshal(deviceGroupUpsert{
		ID:          id,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.getHostURL("/device_group/upsert"), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}

func (c *Client) DeleteDeviceGroup(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.getHostURL(fmt.Sprintf("/device_group/%s", id)), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	return err
}
